package provider_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

func windowLabels(windows []usage.Window) []string {
	labels := make([]string, len(windows))
	for i, w := range windows {
		labels[i] = w.Label
	}
	return labels
}

func TestDefaultRegistrationsWindowsMatchFetchedLabels(t *testing.T) {
	// NewClaude は実ホームの .claude.json を読むため、Claude の Fetch だけは同じ形式の一時ファイルで実行する。
	fetchers := map[string]provider.Provider{
		"Claude": newClaudeAt(writeClaudeJSON(t, validCache), fetchedAt, time.Hour),
	}
	want := map[string][]string{
		"Claude": {"5h", "Weekly"},
		"Codex":  {"5h", "Weekly"},
		"Cursor": {"Monthly"},
	}

	registrations := provider.DefaultRegistrations()

	if len(registrations) != len(want) {
		t.Fatalf("len(DefaultRegistrations()) = %d, want %d", len(registrations), len(want))
	}
	for _, r := range registrations {
		name := r.Provider.Name()
		t.Run(name, func(t *testing.T) {
			if !slices.Equal(r.Windows, want[name]) {
				t.Errorf("Windows = %v, want %v", r.Windows, want[name])
			}
			fetcher, ok := fetchers[name]
			if !ok {
				fetcher = r.Provider
			}
			got, err := fetcher.Fetch(context.Background())
			if err != nil {
				t.Fatalf("Fetch() error = %v", err)
			}
			if fetched := windowLabels(got.Windows); !slices.Equal(fetched, r.Windows) {
				t.Errorf("Fetch labels = %v, registration Windows = %v", fetched, r.Windows)
			}
		})
	}
}
