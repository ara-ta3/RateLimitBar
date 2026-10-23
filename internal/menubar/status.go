package menubar

import (
	"fmt"
	"strings"

	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/popover"
	"ratelimitbar/internal/usage"
)

func statusModel(results []usage.Result, sel *Selection, language i18n.Language) popover.Status {
	model := popover.Status{Tooltip: "RateLimitBar — " + language.Text("Used percentage", "使用済みの割合")}
	for _, result := range results {
		if !sel.anyEnabled(result.Provider) {
			continue
		}
		entry := popover.StatusProvider{Label: result.Provider}
		var selected []string
		if result.Err == nil {
			for _, window := range result.Usage.Windows {
				if !sel.Enabled(result.Provider, window.Label) {
					continue
				}
				label := windowLabel(language, window.Label)
				entry.Windows = append(entry.Windows, popover.StatusWindow{Label: label, UsedPercent: window.UsedPercent, Stale: result.Usage.Stale})
				selected = append(selected, fmt.Sprintf("%s %d%%", label, window.UsedPercent))
			}
		}
		switch {
		case result.Err != nil:
			entry.Unavailable = true
			model.Tooltip += "\n" + result.Provider + ": " + language.Text(failedToFetch, "取得に失敗しました")
		case len(selected) == 0:
			entry.Unavailable = true
			model.Tooltip += "\n" + result.Provider + ": " + language.Text("No usage data", "使用率のデータがありません")
		default:
			model.Tooltip += "\n" + result.Provider + ": " + strings.Join(selected, " / ")
			if result.Usage.Stale {
				model.Tooltip += language.Text(staleSuffix, "（古いデータ）")
			}
		}
		model.Providers = append(model.Providers, entry)
	}
	return model
}
