package provider

// Registration は、Provider と、その Provider が持つ Window のラベル列を対にしたもの。
// Windows は固定のラベルだけを宣言する。取得して初めて決まるラベル(Codex のモデル固有 limit)は含まない。
type Registration struct {
	Provider Provider
	Windows  []string
}

func DefaultRegistrations(autoRefreshClaude func() bool) []Registration {
	return []Registration{
		{NewClaude(autoRefreshClaude), claudeWindowLabels()},
		{NewCodex(), codexWindowLabels()},
		{NewCursor(), []string{WindowMonthly}},
	}
}
