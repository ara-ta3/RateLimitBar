package provider

import (
	"context"

	"ratelimitbar/internal/usage"
)

type Provider interface {
	Name() string
	Fetch(ctx context.Context) (usage.Usage, error)
}

// Window のラベル。取得処理とメニューバーの切り替え項目で共有する。
const (
	WindowFiveHour = "5h"
	WindowWeekly   = "Weekly"
	WindowMonthly  = "Monthly"
)
