package provider

import (
	"context"

	"ratelimitbar/internal/usage"
)

type Provider interface {
	Name() string
	// 更新失敗時も、表示できる保存済みデータがあれば error と併せて返す。
	Fetch(ctx context.Context) (usage.Usage, error)
}

// Window のラベル。取得処理とメニューバーの切り替え項目で共有する。
const (
	WindowFiveHour = "5h"
	WindowWeekly   = "Weekly"
	WindowMonthly  = "Monthly"
)
