package menubar

import (
	"fmt"
	"strings"

	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

const (
	failedToFetch = "Failed to fetch"
	staleSuffix   = " (stale)"
)

// FormatResult はメニュー1行分の表示文字列を作る。
func FormatResult(r usage.Result) string {
	if r.Err != nil {
		return fmt.Sprintf("%s: %s", r.Provider, failedToFetch)
	}
	windows := make([]string, len(r.Usage.Windows))
	for i, w := range r.Usage.Windows {
		windows[i] = fmt.Sprintf("%s %d%%", w.Label, w.UsedPercent)
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
	var parts []string
	for _, r := range results {
		if part := formatTitlePart(r, sel); part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return fixedTitle
	}
	return strings.Join(parts, titleProviderSep)
}

func formatTitlePart(r usage.Result, sel *Selection) string {
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
		windows = append(windows, fmt.Sprintf("%s %d%%", label, w.UsedPercent))
	}
	if len(windows) == 0 {
		return ""
	}
	return r.Provider + " " + strings.Join(windows, titleWindowSep)
}
