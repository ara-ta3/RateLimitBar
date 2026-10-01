package menubar

import (
	"fmt"
	"strings"

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
