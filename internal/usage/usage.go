package usage

import "time"

// Window は 5h / Weekly / Monthly など、1つの制限期間の使用率を表す。
type Window struct {
	Label       string
	UsedPercent int
	ResetsAt    time.Time
}

// Usage は1つの Provider が持つ複数の Window をまとめる。
type Usage struct {
	Windows []Window
	// Stale は、元データが古く現在の使用率と異なる可能性があることを表す。
	Stale bool
}

// Result は Provider ごとの取得結果で、成功した Usage か Err のどちらかを持つ。
type Result struct {
	Provider string
	Usage    Usage
	Err      error
}
