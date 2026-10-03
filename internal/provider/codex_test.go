package provider_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

// codexRateLimitsResultBody は読みやすさのために整形してある。app-server は1行で返すため、送信前に圧縮する。
const codexRateLimitsResultBody = `{"rateLimits":{"limitId":"codex",
	"primary":{"usedPercent":18,"windowDurationMins":300,"resetsAt":1790882763},
	"secondary":{"usedPercent":20,"windowDurationMins":10080,"resetsAt":1791079296}},
"rateLimitsByLimitId":{
	"codex":{"limitId":"codex",
		"primary":{"usedPercent":18,"windowDurationMins":300},
		"secondary":{"usedPercent":20,"windowDurationMins":10080}},
	"codex_spark":{"limitId":"codex_spark","limitName":"GPT-5.3-Codex-Spark",
		"primary":{"usedPercent":5,"windowDurationMins":300}}}}`

func compactJSON(t string) string {
	var b bytes.Buffer
	if err := json.Compact(&b, []byte(t)); err != nil {
		panic(err)
	}
	return b.String()
}

type rpcMessage struct {
	ID     *json.Number    `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// fakeCodex は codex app-server の代わりに、パイプ越しに JSON-RPC を話す。
// 実際の CommandContext と同様に、ctx が終了したら処理を打ち切る。
type fakeCodex struct {
	// respond は受け取ったメッセージへ返す行を返す。nil なら何も返さない。
	respond func(m rpcMessage) []string

	mu       sync.Mutex
	received []rpcMessage
	launches int
	stops    int
}

func (f *fakeCodex) launch(ctx context.Context) (provider.CodexProcess, error) {
	f.mu.Lock()
	f.launches++
	f.mu.Unlock()

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	var once sync.Once
	kill := func() {
		once.Do(func() {
			inW.Close()
			outW.Close()
		})
	}
	go func() {
		<-ctx.Done()
		kill()
	}()
	go func() {
		scanner := bufio.NewScanner(inR)
		for scanner.Scan() {
			var m rpcMessage
			if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
				continue
			}
			f.mu.Lock()
			f.received = append(f.received, m)
			f.mu.Unlock()
			if f.respond == nil {
				continue
			}
			for _, line := range f.respond(m) {
				if _, err := io.WriteString(outW, line+"\n"); err != nil {
					return
				}
			}
		}
	}()
	return provider.CodexProcess{
		Stdin:  inW,
		Stdout: outR,
		Stop: func() error {
			f.mu.Lock()
			f.stops++
			f.mu.Unlock()
			kill()
			return nil
		},
	}, nil
}

func (f *fakeCodex) methods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	methods := make([]string, len(f.received))
	for i, m := range f.received {
		methods[i] = m.Method
	}
	return methods
}

func (f *fakeCodex) counts() (launches, stops int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.launches, f.stops
}

func idLine(id *json.Number, body string) string {
	return `{"id":` + id.String() + `,` + body + `}`
}

// healthyCodex は initialize と rateLimits/read に正しく応答する。
// 実際の app-server と同様に、id のない通知と、別 id の応答も混ぜる。
func healthyCodex() *fakeCodex {
	return &fakeCodex{respond: func(m rpcMessage) []string {
		switch m.Method {
		case "initialize":
			return []string{idLine(m.ID, `"result":{"userAgent":"x","codexHome":"/h"}`)}
		case "account/rateLimits/read":
			return []string{
				`{"method":"account/updated","params":{"authMode":"chatgpt"}}`,
				`{"id":999,"result":{"rateLimits":null}}`,
				idLine(m.ID, `"result":`+compactJSON(codexRateLimitsResultBody)),
			}
		}
		return nil
	}}
}

func fetchWithin(t *testing.T, p provider.Provider, ctx context.Context) (usage.Usage, error) {
	t.Helper()
	type outcome struct {
		u   usage.Usage
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		u, err := p.Fetch(ctx)
		done <- outcome{u, err}
	}()
	select {
	case o := <-done:
		return o.u, o.err
	case <-time.After(3 * time.Second):
		t.Fatal("Fetch did not return in time")
		return usage.Usage{}, nil
	}
}

func TestCodexNameIsCodex(t *testing.T) {
	if got := provider.NewCodexWithLauncher(healthyCodex().launch, time.Second).Name(); got != "Codex" {
		t.Errorf("Name() = %q, want Codex", got)
	}
}

func TestCodexFetchSendsInitializeInitializedThenRateLimitsRead(t *testing.T) {
	fake := healthyCodex()
	p := provider.NewCodexWithLauncher(fake.launch, time.Second)

	if _, err := fetchWithin(t, p, context.Background()); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	want := []string{"initialize", "initialized", "account/rateLimits/read"}
	if got := fake.methods(); !slices.Equal(got, want) {
		t.Errorf("sent methods = %v, want %v", got, want)
	}
	for _, m := range fake.received {
		isNotification := m.Method == "initialized"
		if isNotification == (m.ID != nil) {
			t.Errorf("%s: id present = %v, want notifications without id and requests with id", m.Method, m.ID != nil)
		}
	}
}

func TestCodexFetchReturnsWindowsFromTheResponseMatchingItsRequestId(t *testing.T) {
	p := provider.NewCodexWithLauncher(healthyCodex().launch, time.Second)

	got, err := fetchWithin(t, p, context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	want := []usage.Window{
		{Label: "5h", UsedPercent: 18, ResetsAt: time.Unix(1790882763, 0)},
		{Label: "Weekly", UsedPercent: 20, ResetsAt: time.Unix(1791079296, 0)},
		{Label: "GPT-5.3-Codex-Spark 5h", UsedPercent: 5},
	}
	if !slices.Equal(got.Windows, want) {
		t.Errorf("Windows = %v, want %v", got.Windows, want)
	}
	if got.Stale {
		t.Error("Stale = true, want false for live data")
	}
}

func TestCodexFetchStopsTheProcessAfterSuccess(t *testing.T) {
	fake := healthyCodex()
	p := provider.NewCodexWithLauncher(fake.launch, time.Second)

	if _, err := fetchWithin(t, p, context.Background()); err != nil {
		t.Fatal(err)
	}

	if launches, stops := fake.counts(); launches != 1 || stops != 1 {
		t.Errorf("launches = %d, stops = %d, want 1 and 1", launches, stops)
	}
}

func TestCodexFetchLaunchesAFreshProcessForEveryFetch(t *testing.T) {
	fake := healthyCodex()
	p := provider.NewCodexWithLauncher(fake.launch, time.Second)

	for range 3 {
		if _, err := fetchWithin(t, p, context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if launches, stops := fake.counts(); launches != 3 || stops != 3 {
		t.Errorf("launches = %d, stops = %d, want 3 and 3 (no resident process)", launches, stops)
	}
}

func TestCodexFetchReturnsErrorAndStopsProcessWhenNotLoggedIn(t *testing.T) {
	fake := &fakeCodex{respond: func(m rpcMessage) []string {
		switch m.Method {
		case "initialize":
			return []string{idLine(m.ID, `"result":{}`)}
		case "account/rateLimits/read":
			return []string{idLine(m.ID, `"error":{"code":-32600,"message":"codex account authentication required to read rate limits"}`)}
		}
		return nil
	}}
	p := provider.NewCodexWithLauncher(fake.launch, time.Second)

	_, err := fetchWithin(t, p, context.Background())

	if err == nil {
		t.Fatal("Fetch() error = nil, want error")
	}
	if _, stops := fake.counts(); stops != 1 {
		t.Errorf("stops = %d, want 1", stops)
	}
}

func TestCodexFetchReturnsErrorWhenInitializeFails(t *testing.T) {
	fake := &fakeCodex{respond: func(m rpcMessage) []string {
		if m.Method == "initialize" {
			return []string{idLine(m.ID, `"error":{"code":-32600,"message":"bad"}`)}
		}
		return nil
	}}
	p := provider.NewCodexWithLauncher(fake.launch, time.Second)

	if _, err := fetchWithin(t, p, context.Background()); err == nil {
		t.Fatal("Fetch() error = nil, want error")
	}
	if _, stops := fake.counts(); stops != 1 {
		t.Errorf("stops = %d, want 1", stops)
	}
}

func TestCodexFetchReturnsLaunchErrorWithoutStopping(t *testing.T) {
	launchErr := errors.New("codex not found")
	p := provider.NewCodexWithLauncher(func(context.Context) (provider.CodexProcess, error) {
		return provider.CodexProcess{}, launchErr
	}, time.Second)

	_, err := fetchWithin(t, p, context.Background())

	if !errors.Is(err, launchErr) {
		t.Errorf("Fetch() error = %v, want it to wrap %v", err, launchErr)
	}
}

func TestCodexFetchTimesOutAndStopsProcessWhenServerNeverAnswers(t *testing.T) {
	fake := &fakeCodex{}
	p := provider.NewCodexWithLauncher(fake.launch, 100*time.Millisecond)

	_, err := fetchWithin(t, p, context.Background())

	if err == nil {
		t.Fatal("Fetch() error = nil, want a timeout error")
	}
	if _, stops := fake.counts(); stops != 1 {
		t.Errorf("stops = %d, want 1", stops)
	}
}

func TestCodexFetchStopsProcessWhenContextIsCanceled(t *testing.T) {
	fake := &fakeCodex{}
	p := provider.NewCodexWithLauncher(fake.launch, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := fetchWithin(t, p, ctx)

	if err == nil {
		t.Fatal("Fetch() error = nil, want an error after cancel")
	}
	if _, stops := fake.counts(); stops != 1 {
		t.Errorf("stops = %d, want 1", stops)
	}
}

func TestCodexFetchWithoutCodexOnPathReturnsError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := fetchWithin(t, provider.NewCodex(), context.Background())

	if err == nil {
		t.Fatal("Fetch() error = nil, want an error when codex is not on PATH")
	}
}

func writeCodexExecutable(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codex with spaces")
	script := `#!/bin/sh
printf '%s' "$0" > "$CODEX_TEST_EXECUTABLE"
while IFS= read -r request; do
    case "$request" in
        *'"method":"initialize"'*) printf '%s\n' '{"id":1,"result":{}}' ;;
        *'"method":"account/rateLimits/read"'*) printf '%s\n' '{"id":2,"result":` + compactJSON(codexRateLimitsResultBody) + `}' ;;
    esac
done
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCodexFetchUsesSelectedExecutableWithoutCLIOnPath(t *testing.T) {
	path := writeCodexExecutable(t)
	marker := filepath.Join(t.TempDir(), "executed")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CODEX_TEST_EXECUTABLE", marker)
	p := provider.NewCodexWithPath(func() string { return path })

	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []usage.Window{
		{Label: "5h", UsedPercent: 18, ResetsAt: time.Unix(1790882763, 0)},
		{Label: "Weekly", UsedPercent: 20, ResetsAt: time.Unix(1791079296, 0)},
		{Label: "GPT-5.3-Codex-Spark 5h", UsedPercent: 5},
	}
	if !slices.Equal(got.Windows, want) {
		t.Fatalf("Windows = %+v, want %+v", got.Windows, want)
	}
	executed, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(executed) != path {
		t.Fatalf("executed = %q, want %q", executed, path)
	}
}

func TestCodexUsesNewSelectionOnNextFetch(t *testing.T) {
	path := writeCodexExecutable(t)
	marker := filepath.Join(t.TempDir(), "executed")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CODEX_TEST_EXECUTABLE", marker)
	p := provider.NewCodexWithPath(func() string { return path })
	if _, err := p.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	path = writeCodexExecutable(t)
	if _, err := p.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	executed, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(executed) != path {
		t.Fatalf("executed after reselection = %q, want %q", executed, path)
	}
}
