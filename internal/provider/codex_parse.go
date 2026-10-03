package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"ratelimitbar/internal/usage"
)

type codexRateWindow struct {
	ResetsAt           *int64  `json:"resetsAt"`
	UsedPercent        float64 `json:"usedPercent"`
	WindowDurationMins *int    `json:"windowDurationMins"`
}

type codexLimit struct {
	LimitID   string           `json:"limitId"`
	LimitName *string          `json:"limitName"`
	Primary   *codexRateWindow `json:"primary"`
	Secondary *codexRateWindow `json:"secondary"`
}

type codexRateLimitsResult struct {
	RateLimits          *codexLimit           `json:"rateLimits"`
	RateLimitsByLimitID map[string]codexLimit `json:"rateLimitsByLimitId"`
}

// codexWindowOrder は、1つの limit 内で Window を並べる表示順。
var codexWindowOrder = []string{WindowFiveHour, WindowWeekly}

func codexWindowLabels() []string { return slices.Clone(codexWindowOrder) }

// windowLabelForMinutes は windowDurationMins から Window のラベルを決める。対象外の長さは false を返す。
func windowLabelForMinutes(minutes int) (string, bool) {
	switch minutes {
	case 300:
		return WindowFiveHour, true
	case 10080:
		return WindowWeekly, true
	}
	return "", false
}

// parseCodexRateLimits は account/rateLimits/read の result から Window を作る。
// 一般の limit を先に、モデル固有の limit を limitId 順に続ける。
func parseCodexRateLimits(result []byte) ([]usage.Window, error) {
	var parsed codexRateLimitsResult
	if err := json.Unmarshal(result, &parsed); err != nil {
		return nil, fmt.Errorf("parse codex rate limits: %w", err)
	}
	if parsed.RateLimits == nil {
		return nil, errors.New("codex rate limits response has no rateLimits")
	}

	windows := limitWindows(*parsed.RateLimits, "")
	ids := make([]string, 0, len(parsed.RateLimitsByLimitID))
	for id := range parsed.RateLimitsByLimitID {
		if id != parsed.RateLimits.LimitID {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		limit := parsed.RateLimitsByLimitID[id]
		name := id
		if limit.LimitName != nil && *limit.LimitName != "" {
			name = *limit.LimitName
		}
		windows = append(windows, limitWindows(limit, name+" ")...)
	}
	return windows, nil
}

func limitWindows(limit codexLimit, labelPrefix string) []usage.Window {
	windowsByLabel := map[string]usage.Window{}
	for _, w := range []*codexRateWindow{limit.Primary, limit.Secondary} {
		if w == nil || w.WindowDurationMins == nil {
			continue
		}
		label, ok := windowLabelForMinutes(*w.WindowDurationMins)
		if !ok {
			continue
		}
		if _, seen := windowsByLabel[label]; !seen {
			window := usage.Window{Label: strings.TrimSpace(labelPrefix + label), UsedPercent: int(math.Round(w.UsedPercent))}
			if w.ResetsAt != nil && *w.ResetsAt > 0 {
				window.ResetsAt = time.Unix(*w.ResetsAt, 0)
			}
			windowsByLabel[label] = window
		}
	}
	var windows []usage.Window
	for _, label := range codexWindowOrder {
		if window, ok := windowsByLabel[label]; ok {
			windows = append(windows, window)
		}
	}
	return windows
}
