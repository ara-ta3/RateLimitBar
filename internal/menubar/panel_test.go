package menubar

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/popover"
	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

func TestPanelShowsLoadingBeforeFirstFetch(t *testing.T) {
	sources := []Source{{Name: "Codex", Windows: []string{"5h"}}}
	panel := newUsagePanel(sources, NewSelection(sources))

	actual := panel.model(i18n.Japanese, nil, time.Time{})

	expected := []popover.Provider{{Name: "Codex", Message: "読み込み中…"}}
	if !reflect.DeepEqual(actual.Providers, expected) {
		t.Fatalf("providers = %#v, want %#v", actual.Providers, expected)
	}
}

func TestPanelShowsUsedAndRemainingPercentagesResetAndSelection(t *testing.T) {
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	sources := []Source{{Name: "Codex", Windows: []string{"5h", provider.WindowWeekly}}}
	sel := NewSelection(sources)
	sel.Toggle("Codex", provider.WindowWeekly)
	panel := newUsagePanel(sources, sel)
	panel.window("Codex", "5h", make(chan struct{}))
	panel.window("Codex", provider.WindowWeekly, make(chan struct{}))
	results := []usage.Result{{Provider: "Codex", Usage: usage.Usage{Windows: []usage.Window{
		{Label: "5h", UsedPercent: 61, ResetsAt: now.Add(130 * time.Minute)},
		{Label: provider.WindowWeekly, UsedPercent: 37},
	}}}}

	actual := panel.model(i18n.Japanese, results, now)

	expected := []popover.Provider{{Name: "Codex", Windows: []popover.Window{
		{Label: "5h", UsedPercent: 61, Usage: "使用済み 61%", Remaining: "残り 39%", Reset: "リセットまで 2h10m", Selected: true, Action: "0", Tooltip: "メニューバーに表示"},
		{Label: "週間", UsedPercent: 37, Usage: "使用済み 37%", Remaining: "残り 63%", Reset: "リセット時刻は不明です", Selected: false, Action: "1", Tooltip: "メニューバーに表示"},
	}}}
	if !reflect.DeepEqual(actual.Providers, expected) {
		t.Fatalf("providers = %#v, want %#v", actual.Providers, expected)
	}
}

func TestPanelReplacesPreviousBarsWithFetchError(t *testing.T) {
	sources := []Source{{Name: "Codex"}}
	panel := newUsagePanel(sources, NewSelection(sources))
	panel.model(i18n.Japanese, []usage.Result{{Provider: "Codex", Usage: usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 61}}}}}, time.Time{})

	actual := panel.model(i18n.Japanese, []usage.Result{{Provider: "Codex", Err: errors.New("private detail")}}, time.Time{})

	expected := []popover.Provider{{Name: "Codex", Message: "Codex: 取得に失敗しました"}}
	if !reflect.DeepEqual(actual.Providers, expected) {
		t.Fatalf("providers = %#v, want %#v", actual.Providers, expected)
	}
}

func TestPanelMarksStaleUsageInEnglish(t *testing.T) {
	sources := []Source{{Name: "Claude"}}
	panel := newUsagePanel(sources, NewSelection(sources))
	panel.window("Claude", provider.WindowWeekly, make(chan struct{}))

	actual := panel.model(i18n.English, []usage.Result{{Provider: "Claude", Usage: usage.Usage{Stale: true, Windows: []usage.Window{{Label: provider.WindowWeekly, UsedPercent: 42}}}}}, time.Time{})

	expected := []popover.Provider{{Name: "Claude", Message: "Stale data", Windows: []popover.Window{
		{Label: "Weekly", UsedPercent: 42, Usage: "Used 42%", Remaining: "58% left", Reset: "Reset time unavailable", Selected: true, Action: "0", Tooltip: "Show in menu bar"},
	}}}
	if !reflect.DeepEqual(actual.Providers, expected) {
		t.Fatalf("providers = %#v, want %#v", actual.Providers, expected)
	}
}

func TestPanelDoesNotPresentMissingUsageAsZeroPercent(t *testing.T) {
	sources := []Source{{Name: "Cursor"}}
	panel := newUsagePanel(sources, NewSelection(sources))

	actual := panel.model(i18n.Japanese, []usage.Result{{Provider: "Cursor"}}, time.Time{})

	expected := []popover.Provider{{Name: "Cursor", Message: "使用率のデータがありません"}}
	if !reflect.DeepEqual(actual.Providers, expected) {
		t.Fatalf("providers = %#v, want %#v", actual.Providers, expected)
	}
}

func TestPanelIncludesNewlyDiscoveredWindow(t *testing.T) {
	sources := []Source{{Name: "Codex", Windows: []string{"5h"}}}
	sel := NewSelection(sources)
	panel := newUsagePanel(sources, sel)
	sel.declare("Codex", "Spark 5h")
	panel.window("Codex", "Spark 5h", make(chan struct{}))

	actual := panel.model(i18n.English, []usage.Result{{Provider: "Codex", Usage: usage.Usage{Windows: []usage.Window{{Label: "Spark 5h", UsedPercent: 30}}}}}, time.Time{})

	expected := []popover.Provider{{Name: "Codex", Windows: []popover.Window{
		{Label: "Spark 5h", UsedPercent: 30, Usage: "Used 30%", Remaining: "70% left", Reset: "Reset time unavailable", Selected: true, Action: "0", Tooltip: "Show in menu bar"},
	}}}
	if !reflect.DeepEqual(actual.Providers, expected) {
		t.Fatalf("providers = %#v, want %#v", actual.Providers, expected)
	}
}

func TestPanelDispatchesWindowClickToExistingSelectionHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	panel := newUsagePanel(nil, NewSelection(nil))
	clicked := make(chan struct{})
	panel.window("Codex", "5h", clicked)
	events := make(chan string, 1)
	finished := make(chan struct{})
	go func() {
		panel.dispatch(ctx, events, func() {})
		close(finished)
	}()

	events <- "0"

	select {
	case <-clicked:
	case <-time.After(time.Second):
		t.Fatal("panel click did not reach the registered window handler")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop after cancellation")
	}
}

func TestPanelOpeningRedrawsCountdownWithoutFetching(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	panel := newUsagePanel(nil, NewSelection(nil))
	events := make(chan string, 1)
	redrawn := make(chan struct{}, 1)
	finished := make(chan struct{})
	go func() {
		panel.dispatch(ctx, events, func() { redrawn <- struct{}{} })
		close(finished)
	}()

	events <- "open"

	select {
	case <-redrawn:
	case <-time.After(time.Second):
		t.Fatal("opening the panel did not redraw the cached usage")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop after cancellation")
	}
}
