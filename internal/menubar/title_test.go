package menubar_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ratelimitbar/internal/menubar"
	"ratelimitbar/internal/usage"
)

func allSucceeded() []usage.Result {
	return []usage.Result{
		success("Claude",
			usage.Window{Label: "5h", UsedPercent: 23},
			usage.Window{Label: "Weekly", UsedPercent: 48},
		),
		success("Codex", usage.Window{Label: "5h", UsedPercent: 61}),
		success("Cursor", usage.Window{Label: "Monthly", UsedPercent: 18}),
	}
}

func TestFormatTitleShowsEveryWindowWhenAllEnabled(t *testing.T) {
	sel := menubar.NewSelection(testSources())

	got := menubar.FormatTitle(allSucceeded(), sel)

	want := "Claude 5h 23% / W 48%  Codex 5h 61%  Cursor M 18%"
	if got != want {
		t.Errorf("FormatTitle() = %q, want %q", got, want)
	}
}

func TestFormatTitleOmitsDisabledWindowAndKeepsTheRest(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	sel.Toggle("Claude", "Weekly")

	got := menubar.FormatTitle(allSucceeded(), sel)

	want := "Claude 5h 23%  Codex 5h 61%  Cursor M 18%"
	if got != want {
		t.Errorf("FormatTitle() = %q, want %q", got, want)
	}
}

func TestFormatTitleOmitsProviderWhoseWindowsAreAllDisabled(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	sel.Toggle("Codex", "5h")

	got := menubar.FormatTitle(allSucceeded(), sel)

	want := "Claude 5h 23% / W 48%  Cursor M 18%"
	if got != want {
		t.Errorf("FormatTitle() = %q, want %q", got, want)
	}
}

func TestFormatTitleFallsBackToFixedTitleWhenEverythingIsDisabled(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	for _, src := range testSources() {
		for _, label := range src.Windows {
			sel.Toggle(src.Name, label)
		}
	}

	got := menubar.FormatTitle(allSucceeded(), sel)

	if got != "RateLimit" {
		t.Errorf("FormatTitle() = %q, want %q", got, "RateLimit")
	}
}

func TestFormatTitleIsFixedTitleBeforeAnythingIsFetched(t *testing.T) {
	sel := menubar.NewSelection(testSources())

	for name, results := range map[string][]usage.Result{"nil": nil, "empty": {}} {
		if got := menubar.FormatTitle(results, sel); got != "RateLimit" {
			t.Errorf("FormatTitle(%s results) = %q, want %q", name, got, "RateLimit")
		}
	}
}

func TestFormatTitleShowsSingleFailureMarkerForFailedProviderWithEnabledWindows(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	results := allSucceeded()
	results[0] = usage.Result{Provider: "Claude", Err: errors.New("boom")}

	got := menubar.FormatTitle(results, sel)

	want := "Claude --  Codex 5h 61%  Cursor M 18%"
	if got != want {
		t.Errorf("FormatTitle() = %q, want %q", got, want)
	}
	if n := strings.Count(got, "Claude"); n != 1 {
		t.Errorf("Claude appears %d times in %q, want exactly 1", n, got)
	}
	if strings.Contains(got, "boom") {
		t.Errorf("FormatTitle() = %q, must not expose the error detail", got)
	}
}

func TestFormatTitleShowsFailureMarkerWhenOnlySomeWindowsOfFailedProviderAreEnabled(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	sel.Toggle("Claude", "Weekly")
	results := allSucceeded()
	results[0] = usage.Result{Provider: "Claude", Err: errors.New("boom")}

	got := menubar.FormatTitle(results, sel)

	want := "Claude --  Codex 5h 61%  Cursor M 18%"
	if got != want {
		t.Errorf("FormatTitle() = %q, want %q", got, want)
	}
}

func TestFormatTitleHidesFailedProviderWhoseWindowsAreAllDisabled(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	sel.Toggle("Claude", "5h")
	sel.Toggle("Claude", "Weekly")
	results := allSucceeded()
	results[0] = usage.Result{Provider: "Claude", Err: errors.New("boom")}

	got := menubar.FormatTitle(results, sel)

	want := "Codex 5h 61%  Cursor M 18%"
	if got != want {
		t.Errorf("FormatTitle() = %q, want %q", got, want)
	}
}

func TestFormatTitleShowsFailureMarkerInsteadOfFixedTitleWhenOnlyFailedProviderIsEnabled(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	sel.Toggle("Codex", "5h")
	sel.Toggle("Codex", "Weekly")
	sel.Toggle("Cursor", "Monthly")
	results := allSucceeded()
	results[0] = usage.Result{Provider: "Claude", Err: errors.New("boom")}

	got := menubar.FormatTitle(results, sel)

	if got != "Claude --" {
		t.Errorf("FormatTitle() = %q, want %q", got, "Claude --")
	}
}

func TestFormatTitleDoesNotMarkStaleUsage(t *testing.T) {
	sel := menubar.NewSelection(testSources())
	results := allSucceeded()
	results[0].Usage.Stale = true

	got := menubar.FormatTitle(results, sel)

	want := "Claude 5h 23% / W 48%  Codex 5h 61%  Cursor M 18%"
	if got != want {
		t.Errorf("FormatTitle() = %q, want %q (stale is shown in the dropdown only)", got, want)
	}
}

func newTitleState(item *fakeItem, sel *menubar.Selection) *menubar.TitleState {
	return menubar.NewTitleState(item, sel)
}

func fetchOf(results []usage.Result, err error, calls *int) menubar.Fetch {
	return func(context.Context) ([]usage.Result, error) {
		*calls++
		return results, err
	}
}

func TestTitleStateIsFixedTitleBeforeFirstRefresh(t *testing.T) {
	title := &fakeItem{}
	state := newTitleState(title, menubar.NewSelection(testSources()))

	state.Redraw()

	if title.title != "RateLimit" {
		t.Errorf("title before first refresh = %q, want %q", title.title, "RateLimit")
	}
}

func TestToggleReflectsInAlreadyShownTitleWithoutRefetching(t *testing.T) {
	title := &fakeItem{}
	sel := menubar.NewSelection(testSources())
	state := newTitleState(title, sel)
	var calls int
	fetch := fetchOf(allSucceeded(), nil, &calls)

	if err := menubar.ApplyRefresh(context.Background(), fetch, setters([]*fakeItem{{}, {}, {}}), state); err != nil {
		t.Fatal(err)
	}
	if want := "Claude 5h 23% / W 48%  Codex 5h 61%  Cursor M 18%"; title.title != want {
		t.Fatalf("title after refresh = %q, want %q", title.title, want)
	}

	if sel.Toggle("Claude", "Weekly") {
		t.Fatal("Toggle() = true, want the window turned off")
	}
	state.Redraw()

	if want := "Claude 5h 23%  Codex 5h 61%  Cursor M 18%"; title.title != want {
		t.Errorf("title after toggle = %q, want %q", title.title, want)
	}
	if calls != 1 {
		t.Errorf("fetch called %d times, want 1 (toggle must not refetch)", calls)
	}

	sel.Toggle("Claude", "Weekly")
	state.Redraw()
	if want := "Claude 5h 23% / W 48%  Codex 5h 61%  Cursor M 18%"; title.title != want {
		t.Errorf("title after toggling back on = %q, want %q", title.title, want)
	}
}

func TestToggleToAllDisabledReturnsToFixedTitleOnTheSameTitle(t *testing.T) {
	title := &fakeItem{}
	sel := menubar.NewSelection(testSources())
	state := newTitleState(title, sel)
	var calls int
	fetch := fetchOf(allSucceeded(), nil, &calls)
	if err := menubar.ApplyRefresh(context.Background(), fetch, setters([]*fakeItem{{}, {}, {}}), state); err != nil {
		t.Fatal(err)
	}

	for _, src := range testSources() {
		for _, label := range src.Windows {
			sel.Toggle(src.Name, label)
			state.Redraw()
		}
	}

	if title.title != "RateLimit" {
		t.Errorf("title with everything off = %q, want %q", title.title, "RateLimit")
	}
}

func TestSelectionSurvivesLaterRefreshes(t *testing.T) {
	title := &fakeItem{}
	sel := menubar.NewSelection(testSources())
	state := newTitleState(title, sel)
	rows := setters([]*fakeItem{{}, {}, {}})
	var calls int
	if err := menubar.ApplyRefresh(context.Background(), fetchOf(allSucceeded(), nil, &calls), rows, state); err != nil {
		t.Fatal(err)
	}
	sel.Toggle("Claude", "Weekly")
	state.Redraw()

	updated := allSucceeded()
	updated[0] = success("Claude",
		usage.Window{Label: "5h", UsedPercent: 30},
		usage.Window{Label: "Weekly", UsedPercent: 55},
	)
	if err := menubar.ApplyRefresh(context.Background(), fetchOf(updated, nil, &calls), rows, state); err != nil {
		t.Fatal(err)
	}

	if want := "Claude 5h 30%  Codex 5h 61%  Cursor M 18%"; title.title != want {
		t.Errorf("title after re-fetch = %q, want %q (Weekly must stay off)", title.title, want)
	}
}

func TestToggleAfterFailedRefreshShowsFailureMarkerOnTheSameTitle(t *testing.T) {
	title := &fakeItem{}
	sel := menubar.NewSelection(testSources())
	state := newTitleState(title, sel)
	failed := allSucceeded()
	fetchErr := errors.New("boom")
	failed[0] = usage.Result{Provider: "Claude", Err: fetchErr}
	var calls int

	err := menubar.ApplyRefresh(context.Background(), fetchOf(failed, fetchErr, &calls), setters([]*fakeItem{{}, {}, {}}), state)

	if !errors.Is(err, fetchErr) {
		t.Fatalf("ApplyRefresh() error = %v, want %v", err, fetchErr)
	}
	if want := "Claude --  Codex 5h 61%  Cursor M 18%"; title.title != want {
		t.Fatalf("title after failed refresh = %q, want %q", title.title, want)
	}

	sel.Toggle("Claude", "5h")
	sel.Toggle("Claude", "Weekly")
	state.Redraw()

	if want := "Codex 5h 61%  Cursor M 18%"; title.title != want {
		t.Errorf("title after disabling every Claude window = %q, want %q", title.title, want)
	}
}

func TestApplyRefreshKeepsDropdownRowsWhileTitleHidesDisabledWindow(t *testing.T) {
	title := &fakeItem{}
	sel := menubar.NewSelection(testSources())
	sel.Toggle("Claude", "Weekly")
	state := newTitleState(title, sel)
	rows := []*fakeItem{{}, {}, {}}
	var calls int

	if err := menubar.ApplyRefresh(context.Background(), fetchOf(allSucceeded(), nil, &calls), setters(rows), state); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(rows[0].title, "Weekly 48%") {
		t.Errorf("dropdown row = %q, must still show the disabled window", rows[0].title)
	}
	if strings.Contains(title.title, "48%") {
		t.Errorf("menu bar title = %q, must not show the disabled window", title.title)
	}
}

func TestApplyRefreshCountMismatchLeavesTitleUntouched(t *testing.T) {
	title := &fakeItem{title: "unchanged"}
	state := newTitleState(title, menubar.NewSelection(testSources()))
	var calls int

	err := menubar.ApplyRefresh(context.Background(), fetchOf(allSucceeded(), nil, &calls), setters([]*fakeItem{{}}), state)

	if !errors.Is(err, menubar.ErrResultCountMismatch) {
		t.Errorf("ApplyRefresh() error = %v, want ErrResultCountMismatch", err)
	}
	if title.title != "unchanged" {
		t.Errorf("title = %q, must not be updated on count mismatch", title.title)
	}
}
