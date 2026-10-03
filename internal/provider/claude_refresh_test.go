package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"ratelimitbar/internal/usage"
)

const refreshOldCache = `{"cachedUsageUtilization":{"fetchedAtMs":1700000000000,"utilization":{"five_hour":{"utilization":23},"seven_day":{"utilization":48}}}}`
const refreshNewCache = `{"cachedUsageUtilization":{"fetchedAtMs":1700001200000,"utilization":{"five_hour":{"utilization":70},"seven_day":{"utilization":80}}}}`

func refreshTestClaude(t *testing.T) claude {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(path, []byte(refreshOldCache), 0600); err != nil {
		t.Fatal(err)
	}
	return claude{path: path, staleAfter: 15 * time.Minute, now: func() time.Time { return time.UnixMilli(1700001200000) }}
}

func TestClaudeRefreshesStaleCacheAndReadsUpdatedUsage(t *testing.T) {
	c := refreshTestClaude(t)
	calls := 0
	c.refresh = func(context.Context) error {
		calls++
		return os.WriteFile(c.path, []byte(refreshNewCache), 0600)
	}
	got, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 70}, {Label: "Weekly", UsedPercent: 80}}, Stale: false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Fetch() = %+v, want %+v", got, want)
	}
	if calls != 1 {
		t.Fatalf("refresh calls = %d, want 1", calls)
	}
	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("refresh calls after fresh fetch = %d, want 1", calls)
	}
}

func TestClaudeWithoutAutoRefreshKeepsStaleCache(t *testing.T) {
	c := refreshTestClaude(t)
	got, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 23}, {Label: "Weekly", UsedPercent: 48}}, Stale: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Fetch() = %+v, want %+v", got, want)
	}
}

func TestClaudeReportsAutoRefreshFailure(t *testing.T) {
	c := refreshTestClaude(t)
	refreshErr := errors.New("command failed")
	c.refresh = func(context.Context) error { return refreshErr }
	got, err := c.Fetch(context.Background())
	if !errors.Is(err, refreshErr) {
		t.Fatalf("Fetch() error = %v, want %v", err, refreshErr)
	}
	if got.Stale != true {
		t.Fatalf("Stale = %v, want true", got.Stale)
	}
}

func TestClaudeDoesNotRefreshUnreadableCache(t *testing.T) {
	c := refreshTestClaude(t)
	if err := os.Remove(c.path); err != nil {
		t.Fatal(err)
	}
	c.refresh = func(context.Context) error { t.Fatal("refresh called for unreadable cache"); return nil }
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("Fetch() error = nil, want error")
	}
}

func TestRefreshClaudeCacheRunsUsageCommand(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$CLAUDE_TEST_ARGS\"\npwd -P >> \"$CLAUDE_TEST_ARGS\"\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CLAUDE_TEST_ARGS", marker)
	if err := refreshClaudeCache(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	resolvedTemp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if want := "-p\n/usage\n" + resolvedTemp + "\n"; string(got) != want {
		t.Fatalf("command args and directory = %q, want %q", got, want)
	}
}

func TestRefreshClaudeCacheRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := refreshClaudeCache(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestClaudeRemainsStaleWhenCommandDoesNotUpdateCache(t *testing.T) {
	c := refreshTestClaude(t)
	c.refresh = func(context.Context) error { return nil }
	got, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Stale != true {
		t.Fatalf("Stale = %v, want true", got.Stale)
	}
}

func TestNewClaudeEnablesAutoRefreshOnlyWhenRequested(t *testing.T) {
	disabled := NewClaude(false).(claude)
	if disabled.refresh != nil {
		t.Fatal("auto refresh enabled with false")
	}
	enabled := NewClaude(true).(claude)
	if enabled.refresh == nil {
		t.Fatal("auto refresh disabled with true")
	}
}
