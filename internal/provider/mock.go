package provider

import (
	"context"

	"ratelimitbar/internal/usage"
)

type mock struct {
	name  string
	usage usage.Usage
}

func (m mock) Name() string { return m.name }

func (m mock) Fetch(context.Context) (usage.Usage, error) { return m.usage, nil }

func NewMockCodex() Provider {
	return mock{"Codex", usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 61}, {Label: "Weekly", UsedPercent: 37}}}}
}

func NewMockCursor() Provider {
	return mock{"Cursor", usage.Usage{Windows: []usage.Window{{Label: "Monthly", UsedPercent: 18}}}}
}
