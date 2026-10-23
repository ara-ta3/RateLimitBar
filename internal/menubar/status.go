package menubar

import (
	"fmt"
	"strings"

	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/popover"
	"ratelimitbar/internal/usage"
)

var statusProviderLabels = map[string]string{
	"Claude": "CL",
	"Codex":  "CX",
	"Cursor": "CU",
}

func statusModel(results []usage.Result, sel *Selection, language i18n.Language) popover.Status {
	model := popover.Status{Tooltip: "RateLimitBar — " + language.Text("Used percentage", "使用済みの割合")}
	for _, result := range results {
		if !sel.anyEnabled(result.Provider) {
			continue
		}
		label := statusProviderLabels[result.Provider]
		if label == "" {
			label = result.Provider
		}
		entry := popover.StatusProvider{Label: label, Stale: result.Usage.Stale}
		var selected []string
		for _, window := range result.Usage.Windows {
			if !sel.Enabled(result.Provider, window.Label) {
				continue
			}
			entry.UsedPercent = max(entry.UsedPercent, window.UsedPercent)
			selected = append(selected, fmt.Sprintf("%s %d%%", windowLabel(language, window.Label), window.UsedPercent))
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
			if entry.Stale {
				model.Tooltip += language.Text(staleSuffix, "（古いデータ）")
			}
		}
		model.Providers = append(model.Providers, entry)
	}
	return model
}
