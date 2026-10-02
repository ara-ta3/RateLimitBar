package provider

import (
	"slices"
	"testing"

	"ratelimitbar/internal/usage"
)

func TestWindowLabelForMinutes(t *testing.T) {
	tests := []struct {
		name      string
		minutes   int
		wantLabel string
		wantOK    bool
	}{
		{"300 minutes is 5h", 300, WindowFiveHour, true},
		{"10080 minutes is Weekly", 10080, WindowWeekly, true},
		{"other duration is unknown", 45, "", false},
		{"zero is unknown", 0, "", false},
		{"negative is unknown", -300, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, ok := windowLabelForMinutes(tt.minutes)
			if label != tt.wantLabel || ok != tt.wantOK {
				t.Errorf("windowLabelForMinutes(%d) = (%q, %v), want (%q, %v)", tt.minutes, label, ok, tt.wantLabel, tt.wantOK)
			}
		})
	}
}

func TestParseCodexRateLimits(t *testing.T) {
	tests := []struct {
		name string
		json string
		want []usage.Window
	}{
		{
			name: "primary 300 and secondary 10080 become 5h and Weekly (mirrored in rateLimitsByLimitId without duplicates)",
			json: `{"rateLimits":{"limitId":"codex","limitName":null,
				"primary":{"usedPercent":18,"windowDurationMins":300,"resetsAt":1790882763},
				"secondary":{"usedPercent":20,"windowDurationMins":10080,"resetsAt":1791079296}},
			"rateLimitsByLimitId":{"codex":{"limitId":"codex","limitName":null,
				"primary":{"usedPercent":18,"windowDurationMins":300,"resetsAt":1790882763},
				"secondary":{"usedPercent":20,"windowDurationMins":10080,"resetsAt":1791079296}}}}`,
			want: []usage.Window{{Label: "5h", UsedPercent: 18}, {Label: "Weekly", UsedPercent: 20}},
		},
		{
			name: "rateLimitsByLimitId absent",
			json: `{"rateLimits":{"limitId":"codex",
				"primary":{"usedPercent":18,"windowDurationMins":300},
				"secondary":{"usedPercent":20,"windowDurationMins":10080}}}`,
			want: []usage.Window{{Label: "5h", UsedPercent: 18}, {Label: "Weekly", UsedPercent: 20}},
		},
		{
			name: "label is decided by duration, not by primary/secondary position",
			json: `{"rateLimits":{"limitId":"codex",
				"primary":{"usedPercent":70,"windowDurationMins":10080},
				"secondary":{"usedPercent":9,"windowDurationMins":300}}}`,
			want: []usage.Window{{Label: "5h", UsedPercent: 9}, {Label: "Weekly", UsedPercent: 70}},
		},
		{
			name: "a lone primary of 10080 minutes is Weekly, not 5h",
			json: `{"rateLimits":{"limitId":"codex",
				"primary":{"usedPercent":70,"windowDurationMins":10080},"secondary":null}}`,
			want: []usage.Window{{Label: "Weekly", UsedPercent: 70}},
		},
		{
			name: "window with an unrecognized duration is ignored",
			json: `{"rateLimits":{"limitId":"codex",
				"primary":{"usedPercent":50,"windowDurationMins":45},
				"secondary":{"usedPercent":20,"windowDurationMins":10080}}}`,
			want: []usage.Window{{Label: "Weekly", UsedPercent: 20}},
		},
		{
			name: "window without duration is ignored",
			json: `{"rateLimits":{"limitId":"codex",
				"primary":{"usedPercent":50,"windowDurationMins":null},
				"secondary":{"usedPercent":20,"windowDurationMins":10080}}}`,
			want: []usage.Window{{Label: "Weekly", UsedPercent: 20}},
		},
		{
			name: "model specific limits are added after the general ones with the model name in the label",
			json: `{"rateLimits":{"limitId":"codex",
				"primary":{"usedPercent":18,"windowDurationMins":300},
				"secondary":{"usedPercent":20,"windowDurationMins":10080}},
			"rateLimitsByLimitId":{
				"codex":{"limitId":"codex",
					"primary":{"usedPercent":18,"windowDurationMins":300},
					"secondary":{"usedPercent":20,"windowDurationMins":10080}},
				"codex_spark":{"limitId":"codex_spark","limitName":"GPT-5.3-Codex-Spark",
					"primary":{"usedPercent":5,"windowDurationMins":300},
					"secondary":{"usedPercent":7,"windowDurationMins":10080}}}}`,
			want: []usage.Window{
				{Label: "5h", UsedPercent: 18},
				{Label: "Weekly", UsedPercent: 20},
				{Label: "GPT-5.3-Codex-Spark 5h", UsedPercent: 5},
				{Label: "GPT-5.3-Codex-Spark Weekly", UsedPercent: 7},
			},
		},
		{
			name: "model specific window with an unrecognized duration is ignored",
			json: `{"rateLimits":{"limitId":"codex",
				"primary":{"usedPercent":18,"windowDurationMins":300}},
			"rateLimitsByLimitId":{
				"codex":{"limitId":"codex","primary":{"usedPercent":18,"windowDurationMins":300}},
				"codex_spark":{"limitId":"codex_spark","limitName":"Spark",
					"primary":{"usedPercent":5,"windowDurationMins":45},
					"secondary":{"usedPercent":7,"windowDurationMins":10080}}}}`,
			want: []usage.Window{
				{Label: "5h", UsedPercent: 18},
				{Label: "Spark Weekly", UsedPercent: 7},
			},
		},
		{
			name: "several model specific limits are ordered by limitId so the order is stable",
			json: `{"rateLimits":{"limitId":"codex","primary":{"usedPercent":1,"windowDurationMins":300}},
			"rateLimitsByLimitId":{
				"codex":{"limitId":"codex","primary":{"usedPercent":1,"windowDurationMins":300}},
				"m_b":{"limitId":"m_b","limitName":"Bravo","primary":{"usedPercent":3,"windowDurationMins":300}},
				"m_a":{"limitId":"m_a","limitName":"Alpha","primary":{"usedPercent":2,"windowDurationMins":300}}}}`,
			want: []usage.Window{
				{Label: "5h", UsedPercent: 1},
				{Label: "Alpha 5h", UsedPercent: 2},
				{Label: "Bravo 5h", UsedPercent: 3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCodexRateLimits([]byte(tt.json))
			if err != nil {
				t.Fatalf("parseCodexRateLimits() error = %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseCodexRateLimits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCodexRateLimitsReturnsErrorForUnusableResponse(t *testing.T) {
	tests := map[string]string{
		"invalid json":         `{`,
		"rateLimits is absent": `{"rateLimitsByLimitId":{}}`,
		"empty object":         `{}`,
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseCodexRateLimits([]byte(input)); err == nil {
				t.Error("parseCodexRateLimits() error = nil, want error")
			}
		})
	}
}
