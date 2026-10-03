package menubar

import (
	"fmt"
	"strings"
	"time"

	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

const (
	failedToFetch = "Failed to fetch"
	staleSuffix   = " (stale)"
)

// FormatResult はメニュー1行分の表示文字列を作る。
func FormatResult(r usage.Result) string {
	return formatResultAt(r, time.Now())
}

func formatResultAt(r usage.Result, now time.Time) string {
	if r.Err != nil {
		return fmt.Sprintf("%s: %s", r.Provider, failedToFetch)
	}
	windows := make([]string, len(r.Usage.Windows))
	for i, w := range r.Usage.Windows {
		windows[i] = formatWindow(w, w.Label, now)
	}
	text := fmt.Sprintf("%s: %s", r.Provider, strings.Join(windows, " / "))
	if r.Usage.Stale {
		text += staleSuffix
	}
	return text
}

const (
	fixedTitle         = "RateLimit"
	titleFailureMarker = "--"
	titleProviderSep   = "  "
	titleWindowSep     = " / "
)

var shortWindowLabels = map[string]string{
	provider.WindowWeekly:  "W",
	provider.WindowMonthly: "M",
}

// FormatTitle はメニューバーのタイトルを作る。オンの Window だけを Provider ごとに並べ、
// 表示するものがなければ固定のタイトルを返す。
func FormatTitle(results []usage.Result, sel *Selection) string {
	return formatTitleAt(results, sel, time.Now())
}

func formatTitleAt(results []usage.Result, sel *Selection, now time.Time) string {
	var parts []string
	for _, r := range results {
		if part := formatTitlePart(r, sel, now); part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return fixedTitle
	}
	return strings.Join(parts, titleProviderSep)
}

func formatTitlePart(r usage.Result, sel *Selection, now time.Time) string {
	if r.Err != nil {
		if !sel.anyEnabled(r.Provider) {
			return ""
		}
		return r.Provider + " " + titleFailureMarker
	}
	var windows []string
	for _, w := range r.Usage.Windows {
		if !sel.Enabled(r.Provider, w.Label) {
			continue
		}
		label := w.Label
		if short, ok := shortWindowLabels[label]; ok {
			label = short
		}
		windows = append(windows, formatWindow(w, label, now))
	}
	if len(windows) == 0 {
		return ""
	}
	return r.Provider + " " + strings.Join(windows, titleWindowSep)
}

func formatWindow(w usage.Window, label string, now time.Time) string {
	text := fmt.Sprintf("%s %d%%", label, w.UsedPercent)
	if w.ResetsAt.IsZero() {
		return text
	}
	return text + " (" + formatRemaining(w.ResetsAt.Sub(now)) + ")"
}

func formatRemaining(remaining time.Duration) string {
	if remaining <= 0 {
		return "0m"
	}
	if remaining < time.Minute {
		return "<1m"
	}
	minutes := int64(remaining / time.Minute)
	days, hours, mins := minutes/(24*60), minutes/60%24, minutes%60
	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
