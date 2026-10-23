package menubar

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"fyne.io/systray"

	"ratelimitbar/internal/i18n"
	"ratelimitbar/internal/popover"
	"ratelimitbar/internal/usage"
)

const loadingTitle = "Loading..."

// Run はメニューバーに常駐し、Quit が選ばれるか ctx がキャンセルされるまでブロックする。
// refreshInterval ごとに自動で再取得する。
// 取得は別の goroutine で行い、イベントループをブロックしない。
// 取得 error は表示に反映したうえで onFetchError に渡して継続する。
// 表示を更新できない error は、終了して呼び出し元へ返す。
func Run(ctx context.Context, sources []Source, refreshInterval time.Duration, fetch Fetch, onFetchError func(error), actions []Action, language LanguageSettings, options ...Option) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fatal := make(chan error, 1)
	systray.Run(func() {
		sel := NewSelection(sources)
		title := NewTitleState(systrayTitle{language: language}, sel)
		title.Redraw()
		setters := make([]TitleSetter, len(sources))
		localized := &localizedMenu{language: language, items: setters}
		localized.bind(func(i18n.Language) { title.Redraw() })
		if popover.Available {
			localized.panel = newUsagePanel(sources, sel)
		}
		for i, src := range sources {
			item := systray.AddMenuItem(src.Name+": "+loadingTitle, "")
			item.Disable()
			setters[i] = item
			localized.text(item, src.Name+": Loading...", src.Name+": 読み込み中…")
		}
		systray.AddSeparator()
		for _, src := range sources {
			for _, label := range src.Windows {
				item := systray.AddMenuItemCheckbox(src.Name+" "+label, "", true)
				localized.window(item, src.Name, label)
				if localized.panel != nil {
					localized.panel.window(src.Name, label, item.ClickedCh)
				}
				go toggleOnClick(ctx, item, sel, src.Name, label, title, localized.redraw)
			}
		}
		systray.AddSeparator()
		for _, option := range options {
			item := systray.AddMenuItemCheckbox(option.Title, "", false)
			localized.text(item, option.EnglishTitle, option.Title)
			var enabled atomic.Bool
			if localized.panel != nil {
				localized.panel.command(func(l i18n.Language) popover.Command {
					return popover.Command{Kind: popover.ToggleCommand, Title: l.Text(option.EnglishTitle, option.Title), Checked: enabled.Load()}
				}, item.ClickedCh)
			}
			go toggleOptionOnClick(ctx, item.ClickedCh, item, func(value bool) {
				option.OnChange(value)
				enabled.Store(value)
				localized.redraw()
			})
		}
		if len(options) > 0 {
			systray.AddSeparator()
		}
		triggers := make(chan struct{}, 1)
		triggers <- struct{}{}

		var settingsMenu *systray.MenuItem
		if len(actions) > 0 {
			settingsMenu = systray.AddMenuItem("", "")
			localized.text(settingsMenu, "CLI source settings", "取得元の設定")
		}
		for _, action := range actions {
			item := settingsMenu.AddSubMenuItem(action.Title(), "")
			localized.bind(func(i18n.Language) { item.SetTitle(action.Title()) })
			if localized.panel != nil {
				localized.panel.command(func(i18n.Language) popover.Command {
					detail := ""
					if action.Detail != nil {
						detail = action.Detail()
					}
					return popover.Command{Kind: popover.SourceCommand, Title: action.Title(), Label: action.Label, Detail: detail}
				}, item.ClickedCh)
			}
			go runActionOnClick(ctx, item.ClickedCh, func(ctx context.Context) bool {
				changed := action.OnClick(ctx)
				if changed {
					localized.redraw()
				}
				return changed
			}, triggers)
		}
		if len(actions) > 0 {
			systray.AddSeparator()
		}
		localized.addLanguageMenu(ctx)
		systray.AddSeparator()
		dynamicItems := NewDynamicItems(sel, systrayCheckboxAdder{ctx: ctx, sel: sel, title: title, localized: localized})
		if localized.panel != nil {
			settings := systray.AddMenuItem("", "")
			localized.text(settings, "Settings…", "設定…")
			go runActionOnClick(ctx, settings.ClickedCh, func(context.Context) bool {
				popover.ShowSettings()
				return false
			}, triggers)
		}
		refresh := systray.AddMenuItem("Refresh", "")
		quit := systray.AddMenuItem("Quit", "")
		localized.text(refresh, "Refresh", "更新")
		localized.text(quit, "Quit", "終了")
		if localized.panel != nil {
			localized.panel.refresh = localized.panel.bind(refresh.ClickedCh)
			localized.panel.quit = localized.panel.bind(quit.ClickedCh)
			localized.redraw()
			systray.SetOnTapped(popover.Toggle)
			go localized.panel.dispatch(ctx, popover.Events, localized.redraw)
		}

		fetchWithDynamicItems := func(ctx context.Context) ([]usage.Result, error) {
			results, err := fetch(ctx)
			dynamicItems.Apply(results)
			return results, err
		}
		go refreshWorker(ctx, triggers, fetchWithDynamicItems, setters, title, onFetchError, fatal, localized.update)
		go dispatchEvents(ctx, refresh.ClickedCh, quit.ClickedCh, triggers)
		go periodicTrigger(ctx, refreshInterval, triggers)
	}, cancel)

	select {
	case err := <-fatal:
		return err
	default:
		return nil
	}
}

// systrayCheckboxAdder は、メニュー末尾へチェック項目を追加し、クリックで選択を切り替える。
type systrayCheckboxAdder struct {
	ctx       context.Context
	sel       *Selection
	title     *TitleState
	localized *localizedMenu
}

func (a systrayCheckboxAdder) AddCheckbox(provider, label string) {
	item := systray.AddMenuItemCheckbox(provider+" "+label, "", true)
	a.localized.window(item, provider, label)
	if a.localized.panel != nil {
		a.localized.panel.window(provider, label, item.ClickedCh)
	}
	go toggleOnClick(a.ctx, item, a.sel, provider, label, a.title, a.localized.redraw)
}

type systrayTitle struct{ language LanguageSettings }

func (systrayTitle) SetTitle(title string) { systray.SetTitle(title) }

func (t systrayTitle) SetUsage(results []usage.Result, sel *Selection) {
	if popover.Available {
		popover.UpdateStatus(statusModel(results, sel, t.language.Current()))
		return
	}
	systray.SetTitle(FormatTitle(results, sel))
}

// toggleOnClick は項目のクリックごとに選択を切り替え、チェック表示とタイトルを追従させる。
func toggleOnClick(ctx context.Context, item *systray.MenuItem, sel *Selection, provider, label string, title *TitleState, redraw func()) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-item.ClickedCh:
			if sel.Toggle(provider, label) {
				item.Check()
			} else {
				item.Uncheck()
			}
			title.Redraw()
			redraw()
		}
	}
}

func refreshWorker(ctx context.Context, triggers <-chan struct{}, fetch Fetch, items []TitleSetter, title *TitleState, onFetchError func(error), fatal chan<- error, renderers ...func([]usage.Result)) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-triggers:
		}
		err := ApplyRefresh(ctx, fetch, items, title, renderers...)
		if errors.Is(err, ErrResultCountMismatch) {
			fatal <- err
			systray.Quit()
			return
		}
		if err != nil {
			onFetchError(err)
		}
	}
}

func dispatchEvents(ctx context.Context, refreshClicked, quitClicked <-chan struct{}, triggers chan<- struct{}) {
	defer systray.Quit()
	for {
		select {
		case <-ctx.Done():
			return
		case <-quitClicked:
			return
		case <-refreshClicked:
			select {
			case triggers <- struct{}{}:
			default: // 取得中に再度押された分は、待機中の1件にまとめる
			}
		}
	}
}

// periodicTrigger は interval ごとに triggers へ送る。取得は refreshWorker だけが行うため、手動更新と並行しない。
func periodicTrigger(ctx context.Context, interval time.Duration, triggers chan<- struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case triggers <- struct{}{}:
			default: // 取得待ちが既にある場合は、その1件にまとめる
			}
		}
	}
}
