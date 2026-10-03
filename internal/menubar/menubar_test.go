package menubar_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ratelimitbar/internal/menubar"
	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

func TestFormatResultGuidesUserToCLISettingsWhenExecutableIsUnavailable(t *testing.T) {
	got := menubar.FormatResult(usage.Result{Provider: "Codex", Err: provider.ErrExecutableUnavailable})
	if got != "Codex: CLIが見つかりません（取得元の設定…）" {
		t.Fatalf("FormatResult() = %q", got)
	}
}

func success(name string, windows ...usage.Window) usage.Result {
	return usage.Result{Provider: name, Usage: usage.Usage{Windows: windows}}
}

func TestFormatResultShowsEveryWindowUsage(t *testing.T) {
	got := menubar.FormatResult(success("Claude",
		usage.Window{Label: "5h", UsedPercent: 23},
		usage.Window{Label: "Weekly", UsedPercent: 48},
	))
	for _, want := range []string{"Claude", "5h", "23%", "Weekly", "48%"} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatResult() = %q, want it to contain %q", got, want)
		}
	}
}

func TestFormatResultShowsFailedToFetchForError(t *testing.T) {
	got := menubar.FormatResult(usage.Result{Provider: "Codex", Err: errors.New("boom")})
	if !strings.Contains(got, "Codex") || !strings.Contains(got, "Failed to fetch") {
		t.Errorf("FormatResult() = %q, want provider name and %q", got, "Failed to fetch")
	}
	if strings.Contains(got, "boom") {
		t.Errorf("FormatResult() = %q, must not expose the error detail", got)
	}
}

type fakeItem struct{ title string }

func (f *fakeItem) SetTitle(s string) { f.title = s }

func TestApplyRefreshFailedProviderDoesNotAffectOthers(t *testing.T) {
	items := []*fakeItem{{}, {}, {}}
	fetchErr := errors.New("codex failed")
	fetch := func(context.Context) ([]usage.Result, error) {
		return []usage.Result{
			success("Claude", usage.Window{Label: "5h", UsedPercent: 23}),
			{Provider: "Codex", Err: fetchErr},
			success("Cursor", usage.Window{Label: "Monthly", UsedPercent: 18}),
		}, fetchErr
	}

	err := menubar.ApplyRefresh(context.Background(), fetch, setters(items), freshTitleState())

	if !errors.Is(err, fetchErr) {
		t.Errorf("ApplyRefresh() error = %v, want %v", err, fetchErr)
	}
	if !strings.Contains(items[0].title, "23%") {
		t.Errorf("Claude item = %q", items[0].title)
	}
	if !strings.Contains(items[1].title, "Failed to fetch") {
		t.Errorf("Codex item = %q", items[1].title)
	}
	if !strings.Contains(items[2].title, "18%") {
		t.Errorf("Cursor item = %q", items[2].title)
	}
}

func TestApplyRefreshUpdatesSameItemsAcrossRefreshes(t *testing.T) {
	item := &fakeItem{}
	items := []menubar.TitleSetter{item}
	state := freshTitleState()
	var failing = true
	fetch := func(context.Context) ([]usage.Result, error) {
		if failing {
			err := errors.New("down")
			return []usage.Result{{Provider: "Codex", Err: err}}, err
		}
		return []usage.Result{success("Codex", usage.Window{Label: "5h", UsedPercent: 61})}, nil
	}

	_ = menubar.ApplyRefresh(context.Background(), fetch, items, state)
	if !strings.Contains(item.title, "Failed to fetch") {
		t.Fatalf("before recovery: %q", item.title)
	}

	failing = false
	if err := menubar.ApplyRefresh(context.Background(), fetch, items, state); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(item.title, "Failed to fetch") || !strings.Contains(item.title, "61%") {
		t.Errorf("after recovery the same item must show the new usage, got %q", item.title)
	}
}

func setters(items []*fakeItem) []menubar.TitleSetter {
	out := make([]menubar.TitleSetter, len(items))
	for i, it := range items {
		out[i] = it
	}
	return out
}

func TestApplyRefreshRejectsResultCountMismatchWithoutUpdating(t *testing.T) {
	item := &fakeItem{title: "unchanged"}
	fetch := func(context.Context) ([]usage.Result, error) {
		return []usage.Result{
			success("Claude", usage.Window{Label: "5h", UsedPercent: 23}),
			success("Codex", usage.Window{Label: "5h", UsedPercent: 61}),
		}, nil
	}

	err := menubar.ApplyRefresh(context.Background(), fetch, []menubar.TitleSetter{item}, freshTitleState())

	if !errors.Is(err, menubar.ErrResultCountMismatch) {
		t.Errorf("ApplyRefresh() error = %v, want ErrResultCountMismatch", err)
	}
	if item.title != "unchanged" {
		t.Errorf("item title = %q, must not be updated", item.title)
	}
}

func TestFormatResultMarksStaleUsageAndKeepsValues(t *testing.T) {
	r := success("Claude",
		usage.Window{Label: "5h", UsedPercent: 23},
		usage.Window{Label: "Weekly", UsedPercent: 48},
	)
	r.Usage.Stale = true

	got := menubar.FormatResult(r)

	for _, want := range []string{"5h", "23%", "Weekly", "48%", "(stale)"} {
		if !strings.Contains(got, want) {
			t.Errorf("FormatResult() = %q, want it to contain %q", got, want)
		}
	}
}

func TestFormatResultDoesNotMarkFreshUsageAsStale(t *testing.T) {
	got := menubar.FormatResult(success("Claude", usage.Window{Label: "5h", UsedPercent: 23}))
	if strings.Contains(got, "stale") {
		t.Errorf("FormatResult() = %q, must not contain stale", got)
	}
}

func freshTitleState() *menubar.TitleState {
	return menubar.NewTitleState(&fakeItem{}, menubar.NewSelection(testSources()))
}
