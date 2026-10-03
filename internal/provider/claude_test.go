package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

const fetchedAtMs = 1700000000000

var fetchedAt = time.UnixMilli(fetchedAtMs)

const validCache = `{"other":1,"cachedUsageUtilization":{"fetchedAtMs":1700000000000,` +
	`"utilization":{"five_hour":{"utilization":23.4,"resets_at":"2023-11-15T00:00:00Z"},` +
	`"seven_day":{"utilization":48,"resets_at":"2023-11-20T00:00:00Z"}}}}`

func writeClaudeJSON(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func newClaudeAt(path string, now time.Time, staleAfter time.Duration) provider.Provider {
	return provider.NewClaudeFromFile(path, staleAfter, func() time.Time { return now })
}

func TestClaudeNameIsClaude(t *testing.T) {
	if got := newClaudeAt("unused", fetchedAt, time.Minute).Name(); got != "Claude" {
		t.Errorf("Name() = %q, want Claude", got)
	}
}

func TestClaudeFetchParsesFiveHourAndWeeklyUsage(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, validCache), fetchedAt, time.Hour)

	got, err := p.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	want := []usage.Window{{Label: "5h", UsedPercent: 23, ResetsAt: time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC)}, {Label: "Weekly", UsedPercent: 48, ResetsAt: time.Date(2023, 11, 20, 0, 0, 0, 0, time.UTC)}}
	if len(got.Windows) != len(want) {
		t.Fatalf("Windows = %v, want %v", got.Windows, want)
	}
	for i, w := range want {
		if got.Windows[i] != w {
			t.Errorf("Windows[%d] = %v, want %v", i, got.Windows[i], w)
		}
	}
}

func TestClaudeFetchRoundsFractionalUtilizationToNearest(t *testing.T) {
	cache := `{"cachedUsageUtilization":{"fetchedAtMs":1700000000000,"utilization":{` +
		`"five_hour":{"utilization":23.6},"seven_day":{"utilization":47.5}}}}`
	p := newClaudeAt(writeClaudeJSON(t, cache), fetchedAt, time.Hour)

	got, err := p.Fetch(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if got.Windows[0].UsedPercent != 24 || got.Windows[1].UsedPercent != 48 {
		t.Errorf("Windows = %v, want 24 and 48", got.Windows)
	}
}

func TestClaudeFetchStaleBoundary(t *testing.T) {
	const staleAfter = 15 * time.Minute
	tests := []struct {
		name      string
		elapsed   time.Duration
		wantStale bool
	}{
		{"just fetched", 0, false},
		{"within threshold", staleAfter - time.Millisecond, false},
		{"exactly at threshold", staleAfter, false},
		{"1ms over threshold", staleAfter + time.Millisecond, true},
		{"far over threshold", 24 * time.Hour, true},
	}
	path := writeClaudeJSON(t, validCache)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newClaudeAt(path, fetchedAt.Add(tt.elapsed), staleAfter)

			got, err := p.Fetch(context.Background())

			if err != nil {
				t.Fatalf("Fetch() error = %v", err)
			}
			if got.Stale != tt.wantStale {
				t.Errorf("Stale = %v, want %v", got.Stale, tt.wantStale)
			}
			if len(got.Windows) != 2 {
				t.Errorf("stale usage must keep its windows, got %v", got.Windows)
			}
		})
	}
}

func TestClaudeFetchReturnsErrorForUnreadableInput(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"empty file", ""},
		{"truncated JSON", `{"cachedUsageUtilization":{"fetched`},
		{"not JSON", "hello"},
		{"cachedUsageUtilization missing", `{"other":1}`},
		{"cachedUsageUtilization null", `{"cachedUsageUtilization":null}`},
		{"fetchedAtMs missing", `{"cachedUsageUtilization":{"utilization":{"five_hour":{"utilization":1},"seven_day":{"utilization":2}}}}`},
		{"fetchedAtMs zero", `{"cachedUsageUtilization":{"fetchedAtMs":0,"utilization":{"five_hour":{"utilization":1},"seven_day":{"utilization":2}}}}`},
		{"fetchedAtMs wrong type", `{"cachedUsageUtilization":{"fetchedAtMs":"1700000000000","utilization":{"five_hour":{"utilization":1},"seven_day":{"utilization":2}}}}`},
		{"five_hour missing", `{"cachedUsageUtilization":{"fetchedAtMs":1700000000000,"utilization":{"seven_day":{"utilization":2}}}}`},
		{"seven_day missing", `{"cachedUsageUtilization":{"fetchedAtMs":1700000000000,"utilization":{"five_hour":{"utilization":1}}}}`},
		{"five_hour utilization missing", `{"cachedUsageUtilization":{"fetchedAtMs":1700000000000,"utilization":{"five_hour":{},"seven_day":{"utilization":2}}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newClaudeAt(writeClaudeJSON(t, tt.content), fetchedAt, time.Hour)

			if _, err := p.Fetch(context.Background()); err == nil {
				t.Error("Fetch() error = nil, want error")
			}
		})
	}
}

func TestClaudeFetchReturnsErrorWhenFileDoesNotExist(t *testing.T) {
	p := newClaudeAt(filepath.Join(t.TempDir(), "missing.json"), fetchedAt, time.Hour)

	if _, err := p.Fetch(context.Background()); err == nil {
		t.Error("Fetch() error = nil, want error")
	}
}

func TestClaudeFetchRecoversOnSameProviderAfterFileBecomesValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude.json")
	p := newClaudeAt(path, fetchedAt, time.Hour)

	if _, err := p.Fetch(context.Background()); err == nil {
		t.Fatal("Fetch() before the file exists: error = nil, want error")
	}
	if err := os.WriteFile(path, []byte(`{"cachedUsageUtilization":{"fetched`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Fetch(context.Background()); err == nil {
		t.Fatal("Fetch() with half-written file: error = nil, want error")
	}
	if err := os.WriteFile(path, []byte(validCache), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := p.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() after recovery error = %v", err)
	}
	if got.Windows[0].UsedPercent != 23 {
		t.Errorf("Windows = %v, want fresh values", got.Windows)
	}
}

func TestClaudeFetchRereadsFileOnEveryCall(t *testing.T) {
	path := writeClaudeJSON(t, validCache)
	p := newClaudeAt(path, fetchedAt, time.Hour)
	if _, err := p.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	updated := `{"cachedUsageUtilization":{"fetchedAtMs":1700000000000,"utilization":{` +
		`"five_hour":{"utilization":70},"seven_day":{"utilization":80}}}}`
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := p.Fetch(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if got.Windows[0].UsedPercent != 70 || got.Windows[1].UsedPercent != 80 {
		t.Errorf("Windows = %v, want 70 and 80", got.Windows)
	}
}

func TestDefaultClaudeStaleAfterIsPositive(t *testing.T) {
	if provider.DefaultClaudeStaleAfter <= 0 {
		t.Errorf("DefaultClaudeStaleAfter = %v, want > 0", provider.DefaultClaudeStaleAfter)
	}
}

func TestClaudeFetchAllowsNullResetTime(t *testing.T) {
	cache := `{"cachedUsageUtilization":{"fetchedAtMs":1700000000000,"utilization":{"five_hour":{"utilization":0,"resets_at":null},"seven_day":{"utilization":0,"resets_at":null}}}}`
	p := newClaudeAt(writeClaudeJSON(t, cache), fetchedAt, time.Hour)
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []usage.Window{{Label: "5h", UsedPercent: 0}, {Label: "Weekly", UsedPercent: 0}}
	if len(got.Windows) != len(want) {
		t.Fatalf("Windows = %v, want %v", got.Windows, want)
	}
	if got.Windows[0] != want[0] || got.Windows[1] != want[1] {
		t.Errorf("Windows = %v, want %v", got.Windows, want)
	}
}
