package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"ratelimitbar/internal/claudeusage"
	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

var fetchedAt = time.UnixMilli(1700000000000)

const validCache = `{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{"five_hour":{"used_percentage":23.4,"resets_at":1700006400},"seven_day":{"used_percentage":48,"resets_at":1700438400}}}`

func writeClaudeJSON(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude-usage.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func newClaudeAt(path string, now time.Time, staleAfter time.Duration) provider.Provider {
	return provider.NewClaudeFromFile(path, staleAfter, func() time.Time { return now })
}

func TestClaudeFetchParsesStatusLineCache(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, validCache), fetchedAt, time.Hour)
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, usage.Usage{Windows: []usage.Window{
		{Label: "5h", UsedPercent: 23, ResetsAt: time.Unix(1700006400, 0)},
		{Label: "Weekly", UsedPercent: 48, ResetsAt: time.Unix(1700438400, 0)},
	}, ReceivedAt: fetchedAt.UTC(), Stale: false}) {
		t.Fatalf("Fetch() = %+v", got)
	}
}

func TestClaudeFetchRoundsFractionalUsage(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, `{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{"five_hour":{"used_percentage":23.6,"resets_at":1700006400},"seven_day":{"used_percentage":47.5,"resets_at":1700438400}}}`), fetchedAt, time.Hour)
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Windows, []usage.Window{
		{Label: "5h", UsedPercent: 24, ResetsAt: time.Unix(1700006400, 0)},
		{Label: "Weekly", UsedPercent: 48, ResetsAt: time.Unix(1700438400, 0)},
	}) {
		t.Fatalf("Windows = %+v", got.Windows)
	}
}

func TestClaudeFetchAllowsWeeklyWindowOnly(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, `{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{"seven_day":{"used_percentage":0,"resets_at":1700438400}}}`), fetchedAt, time.Hour)
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, usage.Usage{ReceivedAt: fetchedAt.UTC(), Windows: []usage.Window{{Label: "Weekly", UsedPercent: 0, ResetsAt: time.Unix(1700438400, 0)}}}) {
		t.Fatalf("Fetch() = %+v", got)
	}
}

func TestClaudeFetchAllowsFiveHourWindowOnly(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, `{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{"five_hour":{"used_percentage":100,"resets_at":1700006400}}}`), fetchedAt, time.Hour)
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, usage.Usage{ReceivedAt: fetchedAt.UTC(), Windows: []usage.Window{{Label: "5h", UsedPercent: 100, ResetsAt: time.Unix(1700006400, 0)}}}) {
		t.Fatalf("Fetch() = %+v", got)
	}
}

func TestClaudeFetchMarksOldCacheStale(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, validCache), fetchedAt.Add(15*time.Minute+time.Millisecond), 15*time.Minute)
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Stale != true || len(got.Windows) != 2 {
		t.Fatalf("Fetch() = %+v, want stale usage with both windows", got)
	}
}

func TestClaudeFetchKeepsCacheFreshAtThreshold(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, validCache), fetchedAt.Add(15*time.Minute), 15*time.Minute)
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Stale != false {
		t.Fatalf("Stale = %v, want false", got.Stale)
	}
}

func TestClaudeFetchDropsWindowAtResetTime(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, validCache), time.Unix(1700006400, 0), 15*time.Minute)
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, usage.Usage{Windows: []usage.Window{{Label: "Weekly", UsedPercent: 48, ResetsAt: time.Unix(1700438400, 0)}}, ReceivedAt: fetchedAt.UTC(), Stale: true}) {
		t.Fatalf("Fetch() = %+v", got)
	}
}

func TestClaudeFetchReportsUnavailableWhenAllWindowsExpired(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, validCache), time.Unix(1700438400, 0), time.Hour)
	if _, err := p.Fetch(context.Background()); err == nil {
		t.Fatal("Fetch() error = nil, want no active windows")
	}
}

func TestClaudeFetchReportsUnavailableForEmptyWindows(t *testing.T) {
	p := newClaudeAt(writeClaudeJSON(t, `{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{}}`), fetchedAt, time.Hour)
	if _, err := p.Fetch(context.Background()); err == nil {
		t.Fatal("Fetch() error = nil, want no active windows")
	}
}

func TestClaudeFetchRecoversAfterCacheIsCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-usage.json")
	p := newClaudeAt(path, fetchedAt, time.Hour)
	if _, err := p.Fetch(context.Background()); err == nil {
		t.Fatal("Fetch() error = nil before cache exists")
	}
	if err := os.WriteFile(path, []byte(validCache), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeFetchRereadsUpdatedCache(t *testing.T) {
	path := writeClaudeJSON(t, validCache)
	p := newClaudeAt(path, fetchedAt, time.Hour)
	if _, err := p.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{"seven_day":{"used_percentage":80,"resets_at":1700438400}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Windows, []usage.Window{{Label: "Weekly", UsedPercent: 80, ResetsAt: time.Unix(1700438400, 0)}}) {
		t.Fatalf("Windows = %+v", got.Windows)
	}
}

func TestClaudeDefaultProviderReadsStatusLineCachePath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	path, err := claudeusage.CachePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"fetched_at":"2023-11-14T22:13:20Z","rate_limits":{"five_hour":{"used_percentage":23,"resets_at":4102444800}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := provider.NewClaude().Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Stale != true {
		t.Fatalf("Stale = %v, want true", got.Stale)
	}
}
