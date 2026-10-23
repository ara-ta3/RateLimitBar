package menubar

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/popover"
	"ratelimitbar/internal/usage"
)

func TestStatusUsesHighestSelectedUsageForEachProvider(t *testing.T) {
	selection := NewSelection([]Source{{Name: "Claude", Windows: []string{"5h", "Weekly"}}, {Name: "Codex", Windows: []string{"5h"}}})
	results := []usage.Result{
		{Provider: "Claude", Usage: usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 23}, {Label: "Weekly", UsedPercent: 78}}}},
		{Provider: "Codex", Usage: usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 18}}}},
	}

	actual := statusModel(results, selection, i18n.Japanese)

	expected := popover.Status{
		Providers: []popover.StatusProvider{{Label: "CL", UsedPercent: 78}, {Label: "CX", UsedPercent: 18}},
		Tooltip:   "RateLimitBar — 使用済みの割合\nClaude: 5h 23% / 週間 78%\nCodex: 5h 18%",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("status = %#v, want %#v", actual, expected)
	}
}

func TestStatusDoesNotIncludeDisabledWindowInRepresentativeUsage(t *testing.T) {
	selection := NewSelection([]Source{{Name: "Codex", Windows: []string{"5h", "Weekly"}}})
	selection.Toggle("Codex", "Weekly")

	actual := statusModel([]usage.Result{{Provider: "Codex", Usage: usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 18}, {Label: "Weekly", UsedPercent: 94}}}}}, selection, i18n.English)

	expected := popover.Status{
		Providers: []popover.StatusProvider{{Label: "CX", UsedPercent: 18}},
		Tooltip:   "RateLimitBar — Used percentage\nCodex: 5h 18%",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("status = %#v, want %#v", actual, expected)
	}
}

func TestStatusDoesNotTreatFetchFailureAsZeroUsage(t *testing.T) {
	selection := NewSelection([]Source{{Name: "Cursor", Windows: []string{"Monthly"}}})

	actual := statusModel([]usage.Result{{Provider: "Cursor", Err: errors.New("private error")}}, selection, i18n.Japanese)

	expected := popover.Status{
		Providers: []popover.StatusProvider{{Label: "CU", Unavailable: true}},
		Tooltip:   "RateLimitBar — 使用済みの割合\nCursor: 取得に失敗しました",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("status = %#v, want %#v", actual, expected)
	}
}

func TestStatusMarksStaleDataSeparatelyFromFreshUsage(t *testing.T) {
	selection := NewSelection([]Source{{Name: "Claude", Windows: []string{"5h"}}})

	actual := statusModel([]usage.Result{{Provider: "Claude", Usage: usage.Usage{Stale: true, Windows: []usage.Window{{Label: "5h", UsedPercent: 95}}}}}, selection, i18n.English)

	expected := popover.Status{
		Providers: []popover.StatusProvider{{Label: "CL", UsedPercent: 95, Stale: true}},
		Tooltip:   "RateLimitBar — Used percentage\nClaude: 5h 95% (stale)",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("status = %#v, want %#v", actual, expected)
	}
}

func TestStatusOmitsProvidersWithAllWindowsDisabled(t *testing.T) {
	selection := NewSelection([]Source{{Name: "Codex", Windows: []string{"5h"}}})
	selection.Toggle("Codex", "5h")

	actual := statusModel([]usage.Result{{Provider: "Codex", Err: errors.New("down")}}, selection, i18n.Japanese)

	if !reflect.DeepEqual(actual, popover.Status{Tooltip: "RateLimitBar — 使用済みの割合"}) {
		t.Fatalf("status = %#v, want no provider bars", actual)
	}
}

func TestStatusDistinguishesMissingDataFromMeasuredZero(t *testing.T) {
	selection := NewSelection([]Source{{Name: "Claude", Windows: []string{"5h"}}, {Name: "Codex", Windows: []string{"5h"}}})

	actual := statusModel([]usage.Result{{Provider: "Claude"}, {Provider: "Codex", Usage: usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 0}}}}}, selection, i18n.Japanese)

	expected := popover.Status{
		Providers: []popover.StatusProvider{{Label: "CL", Unavailable: true}, {Label: "CX", UsedPercent: 0}},
		Tooltip:   "RateLimitBar — 使用済みの割合\nClaude: 使用率のデータがありません\nCodex: 5h 0%",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("status = %#v, want %#v", actual, expected)
	}
}

type visualStatusRecorder struct {
	status popover.Status
	text   string
}

func (r *visualStatusRecorder) SetTitle(text string) { r.text = text }
func (r *visualStatusRecorder) SetUsage(results []usage.Result, selection *Selection) {
	r.status = statusModel(results, selection, i18n.English)
}

func TestVisualTitleReflectsSelectionWithoutRefetching(t *testing.T) {
	selection := NewSelection([]Source{{Name: "Codex", Windows: []string{"5h", "Weekly"}}})
	recorder := &visualStatusRecorder{}
	state := NewTitleState(recorder, selection)
	calls := 0
	fetch := func(context.Context) ([]usage.Result, error) {
		calls++
		return []usage.Result{{Provider: "Codex", Usage: usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 18}, {Label: "Weekly", UsedPercent: 94}}}}}, nil
	}
	if err := ApplyRefresh(context.Background(), fetch, []TitleSetter{&localizedTestItem{}}, state); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recorder.status.Providers, []popover.StatusProvider{{Label: "CX", UsedPercent: 94}}) {
		t.Fatalf("status after fetch = %#v", recorder.status)
	}

	selection.Toggle("Codex", "Weekly")
	state.Redraw()

	if !reflect.DeepEqual(recorder.status.Providers, []popover.StatusProvider{{Label: "CX", UsedPercent: 18}}) {
		t.Fatalf("status after toggle = %#v", recorder.status)
	}
	if calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", calls)
	}
	if recorder.text != "" {
		t.Fatalf("text title = %q, want the visual renderer only", recorder.text)
	}
}
