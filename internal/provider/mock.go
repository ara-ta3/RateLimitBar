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

func (m mock) windowLabels() []string {
	labels := make([]string, len(m.usage.Windows))
	for i, w := range m.usage.Windows {
		labels[i] = w.Label
	}
	return labels
}

func newMockCodex() mock {
	return mock{"Codex", usage.Usage{Windows: []usage.Window{{Label: WindowFiveHour, UsedPercent: 61}, {Label: WindowWeekly, UsedPercent: 37}}}}
}

func newMockCursor() mock {
	return mock{"Cursor", usage.Usage{Windows: []usage.Window{{Label: WindowMonthly, UsedPercent: 18}}}}
}

func NewMockCodex() Provider { return newMockCodex() }

func NewMockCursor() Provider { return newMockCursor() }
