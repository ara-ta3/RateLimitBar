package menubar

import (
	"errors"
	"testing"

	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

type localizedTestItem struct{ title string }

func (i *localizedTestItem) SetTitle(title string) { i.title = title }

func TestLanguageSwitchRedrawsCachedResultsAndMenuWithoutFetching(t *testing.T) {
	language := i18n.Japanese
	status := &localizedTestItem{}
	refresh := &localizedTestItem{}
	window := &localizedTestItem{}
	menu := &localizedMenu{
		language: LanguageSettings{Current: func() i18n.Language { return language }},
		items:    []TitleSetter{status},
	}
	menu.text(refresh, "Refresh", "更新")
	menu.window(window, "Claude", provider.WindowWeekly)
	menu.update([]usage.Result{{Provider: "Codex", Err: provider.ErrExecutableUnavailable}})
	if status.title != "Codex: CLIが見つかりません（取得元の設定…）" {
		t.Fatalf("Japanese status = %q", status.title)
	}
	if refresh.title != "更新" {
		t.Fatalf("Japanese refresh = %q", refresh.title)
	}
	if window.title != "Claude 週間" {
		t.Fatalf("Japanese window = %q", window.title)
	}
	language = i18n.English
	menu.redraw()
	if status.title != "Codex: CLI not found (CLI source settings…)" {
		t.Fatalf("English status = %q", status.title)
	}
	if refresh.title != "Refresh" {
		t.Fatalf("English refresh = %q", refresh.title)
	}
	if window.title != "Claude Weekly" {
		t.Fatalf("English window = %q", window.title)
	}
	language = i18n.Japanese
	menu.redraw()
	if refresh.title != "更新" {
		t.Fatalf("switched back refresh = %q", refresh.title)
	}
}

func TestJapaneseFetchFailure(t *testing.T) {
	item := &localizedTestItem{}
	menu := &localizedMenu{language: LanguageSettings{Current: func() i18n.Language { return i18n.Japanese }}, items: []TitleSetter{item}}
	menu.update([]usage.Result{{Provider: "Codex", Err: errors.New("private error detail")}})
	if item.title != "Codex: 取得に失敗しました" {
		t.Fatalf("status = %q", item.title)
	}
}

func TestJapaneseStaleWeeklyResult(t *testing.T) {
	item := &localizedTestItem{}
	menu := &localizedMenu{language: LanguageSettings{Current: func() i18n.Language { return i18n.Japanese }}, items: []TitleSetter{item}}
	menu.update([]usage.Result{{Provider: "Claude", Usage: usage.Usage{Stale: true, Windows: []usage.Window{{Label: provider.WindowWeekly, UsedPercent: 42}}}}})
	if item.title != "Claude: 週間 42%（古いデータ）" {
		t.Fatalf("status = %q", item.title)
	}
}
