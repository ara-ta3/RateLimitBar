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

func TestDefaultRegistrationsDeclareFixedWindowLabels(t *testing.T) {
	want := map[string][]string{
		"Claude": {"5h", "Weekly"},
		"Codex":  {"5h", "Weekly"},
		"Cursor": {"Monthly"},
	}

	registrations := provider.DefaultRegistrations(false)

	if len(registrations) != len(want) {
		t.Fatalf("len(DefaultRegistrations()) = %d, want %d", len(registrations), len(want))
	}
	for _, r := range registrations {
		name := r.Provider.Name()
		if !slices.Equal(r.Windows, want[name]) {
			t.Errorf("%s Windows = %v, want %v", name, r.Windows, want[name])
		}
	}
}

func TestDefaultRegistrationsKeepClaudeWindowsMatchingFetchedLabels(t *testing.T) {
	// NewClaude は実ホームの .claude.json を読むため、Fetch は同じ形式の一時ファイルで行う。
	claude := newClaudeAt(writeClaudeJSON(t, validCache), fetchedAt, time.Hour)
	var claudeWindows []string
	for _, r := range provider.DefaultRegistrations(false) {
		if r.Provider.Name() == "Claude" {
			claudeWindows = r.Windows
		}
	}

	got, err := claude.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if fetched := windowLabels(got.Windows); !slices.Equal(fetched, claudeWindows) {
		t.Errorf("Fetch labels = %v, registration Windows = %v", fetched, claudeWindows)
	}
}
