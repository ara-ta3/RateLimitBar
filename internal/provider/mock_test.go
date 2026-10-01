package provider_test

import (
	"context"
	"testing"

	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

func TestMockProvidersReturnSpecifiedUsage(t *testing.T) {
	tests := []struct {
		name     string
		provider provider.Provider
		wantName string
		want     []usage.Window
	}{
		{"codex", provider.NewMockCodex(), "Codex", []usage.Window{{Label: "5h", UsedPercent: 61}, {Label: "Weekly", UsedPercent: 37}}},
		{"cursor", provider.NewMockCursor(), "Cursor", []usage.Window{{Label: "Monthly", UsedPercent: 18}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.provider.Name(); got != tt.wantName {
				t.Fatalf("Name() = %q, want %q", got, tt.wantName)
			}
			got, err := tt.provider.Fetch(context.Background())
			if err != nil {
				t.Fatalf("Fetch() error = %v", err)
			}
			if len(got.Windows) != len(tt.want) {
				t.Fatalf("Windows = %v, want %v", got.Windows, tt.want)
			}
			for i, w := range tt.want {
				if got.Windows[i] != w {
					t.Errorf("Windows[%d] = %v, want %v", i, got.Windows[i], w)
				}
			}
		})
	}
}
