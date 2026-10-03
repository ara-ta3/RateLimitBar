package menubar

import (
	"testing"
	"time"

	"ratelimitbar/internal/usage"
)

func TestFormatsResetCountdownInTitleAndDropdown(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	result := usage.Result{Provider: "Test", Usage: usage.Usage{Windows: []usage.Window{
		{Label: "5h", UsedPercent: 23, ResetsAt: now.Add(150 * time.Minute)},
		{Label: "Weekly", UsedPercent: 48, ResetsAt: now.Add(76 * time.Hour)},
		{Label: "Monthly", UsedPercent: 18, ResetsAt: now.Add(12 * 24 * time.Hour)},
	}}}
	sel := NewSelection([]Source{{Name: "Test", Windows: []string{"5h", "Weekly", "Monthly"}}})
	if got, want := formatTitleAt([]usage.Result{result}, sel, now), "Test 5h 23% (2h30m) / W 48% (3d4h) / M 18% (12d0h)"; got != want {
		t.Errorf("title = %q, want %q", got, want)
	}
	if got, want := formatResultAt(result, now), "Test: 5h 23% (2h30m) / Weekly 48% (3d4h) / Monthly 18% (12d0h)"; got != want {
		t.Errorf("dropdown = %q, want %q", got, want)
	}
	if got, want := formatTitleAt([]usage.Result{result}, sel, now.Add(time.Minute)), "Test 5h 23% (2h29m) / W 48% (3d3h) / M 18% (11d23h)"; got != want {
		t.Errorf("title after a minute = %q, want %q", got, want)
	}
}

func TestFormatsUnknownResetWithoutCountdown(t *testing.T) {
	now := time.Now()
	got := formatWindow(usage.Window{Label: "5h", UsedPercent: 23}, "5h", now)
	if got != "5h 23%" {
		t.Errorf("window = %q, want 5h 23%%", got)
	}
}

func TestFormatsExpiredResetAsZero(t *testing.T) {
	if got := formatRemaining(-time.Second); got != "0m" {
		t.Errorf("remaining = %q, want 0m", got)
	}
	if got := formatRemaining(0); got != "0m" {
		t.Errorf("at reset = %q, want 0m", got)
	}
}

func TestFormatsLessThanAMinuteWithoutPrematureZero(t *testing.T) {
	if got := formatRemaining(59 * time.Second); got != "<1m" {
		t.Errorf("remaining = %q, want <1m", got)
	}
	if got := formatRemaining(time.Minute); got != "1m" {
		t.Errorf("remaining = %q, want 1m", got)
	}
}
