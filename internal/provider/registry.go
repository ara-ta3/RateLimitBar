package provider

// Registration は、Provider と、その Provider が持つ Window のラベル列を対にしたもの。
// Windows は Provider の Fetch が返すラベルと同じ宣言から得る。
type Registration struct {
	Provider Provider
	Windows  []string
}

func DefaultRegistrations() []Registration {
	codex := newMockCodex()
	cursor := newMockCursor()
	return []Registration{
		{NewClaude(), claudeWindowLabels()},
		{codex, codex.windowLabels()},
		{cursor, cursor.windowLabels()},
	}
}
