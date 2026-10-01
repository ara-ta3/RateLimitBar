package provider

// Registration は、Provider と、その Provider が持つ Window のラベル列を対にしたもの。
// Windows は固定のラベルだけを宣言する。取得して初めて決まるラベル(Codex のモデル固有 limit)は含まない。
type Registration struct {
	Provider Provider
	Windows  []string
}

func DefaultRegistrations() []Registration {
	return []Registration{
		{NewClaude(), claudeWindowLabels()},
		{NewCodex(), codexWindowLabels()},
		{NewCursor(), []string{WindowMonthly}},
	}
}
