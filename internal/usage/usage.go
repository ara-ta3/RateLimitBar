package usage

// Window は 5h / Weekly / Monthly など、1つの制限期間の使用率を表す。
type Window struct {
	Label       string
	UsedPercent int
}

// Usage は1つの Provider が持つ複数の Window をまとめる。
type Usage struct {
	Windows []Window
}

// Result は Provider ごとの取得結果で、成功した Usage か Err のどちらかを持つ。
type Result struct {
	Provider string
	Usage    Usage
	Err      error
}
