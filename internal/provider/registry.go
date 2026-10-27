package provider

// Registration は、Provider と、その Provider が持つ Window のラベル列を対にしたもの。
// Windows は固定のラベルだけを宣言する。取得して初めて決まるラベル(Codex のモデル固有 limit)は含まない。
type Registration struct {
	Provider Provider
	Windows  []string
}

type Config struct {
	CodexPath func() string
}

func DefaultRegistrations(config Config) []Registration {
	return []Registration{
		{NewClaude(), claudeWindowLabels()},
		{NewCodexWithPath(config.CodexPath), codexWindowLabels()},
		{NewCursor(), []string{WindowMonthly}},
	}
}
