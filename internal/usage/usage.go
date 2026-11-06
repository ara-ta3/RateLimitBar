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
	Windows    []Window
	ReceivedAt time.Time
	// Stale は、最終受信から設定時間を超えていることを表す。
	Stale bool
}

// Result は Provider ごとの取得結果。更新失敗時も保存済みの Usage と Err を併せて保持できる。
type Result struct {
	Provider string
	Usage    Usage
	Err      error
}
