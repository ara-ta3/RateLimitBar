package menubar

import (
	"context"
	"errors"
	"time"

	"fyne.io/systray"

	"ratelimitbar/internal/usage"
)

const loadingTitle = "Loading..."

// Run はメニューバーに常駐し、Quit が選ばれるまでブロックする。
// refreshInterval ごとに自動で再取得する。
// 取得は別の goroutine で行い、イベントループをブロックしない。
// 取得 error は表示に反映したうえで onFetchError に渡して継続する。
// 表示を更新できない error は、終了して呼び出し元へ返す。
func Run(sources []Source, refreshInterval time.Duration, fetch Fetch, onFetchError func(error)) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fatal := make(chan error, 1)
	systray.Run(func() {
		sel := NewSelection(sources)
		title := NewTitleState(systrayTitle{}, sel)
		title.Redraw()
		setters := make([]TitleSetter, len(sources))
		for i, src := range sources {
			item := systray.AddMenuItem(src.Name+": "+loadingTitle, "")
			item.Disable()
			setters[i] = item
		}
		systray.AddSeparator()
		for _, src := range sources {
			for _, label := range src.Windows {
				item := systray.AddMenuItemCheckbox(src.Name+" "+label, "", true)
				go toggleOnClick(ctx, item, sel, src.Name, label, title)
			}
		}
		systray.AddSeparator()
		dynamicItems := NewDynamicItems(sel, systrayCheckboxAdder{ctx: ctx, sel: sel, title: title})
		refresh := systray.AddMenuItem("Refresh", "")
		quit := systray.AddMenuItem("Quit", "")

		triggers := make(chan struct{}, 1)
		triggers <- struct{}{}
		fetchWithDynamicItems := func(ctx context.Context) ([]usage.Result, error) {
			results, err := fetch(ctx)
			dynamicItems.Apply(results)
			return results, err
		}
		go refreshWorker(ctx, triggers, fetchWithDynamicItems, setters, title, onFetchError, fatal)
		go dispatchEvents(ctx, refresh.ClickedCh, quit.ClickedCh, triggers)
		go periodicTrigger(ctx, refreshInterval, triggers)
	}, func() {})

	select {
	case err := <-fatal:
		return err
	default:
		return nil
	}
}

// systrayCheckboxAdder は、メニュー末尾へチェック項目を追加し、クリックで選択を切り替える。
type systrayCheckboxAdder struct {
	ctx   context.Context
	sel   *Selection
	title *TitleState
}

func (a systrayCheckboxAdder) AddCheckbox(provider, label string) {
	item := systray.AddMenuItemCheckbox(provider+" "+label, "", true)
	go toggleOnClick(a.ctx, item, a.sel, provider, label, a.title)
}

type systrayTitle struct{}

func (systrayTitle) SetTitle(title string) { systray.SetTitle(title) }

// toggleOnClick は項目のクリックごとに選択を切り替え、チェック表示とタイトルを追従させる。
func toggleOnClick(ctx context.Context, item *systray.MenuItem, sel *Selection, provider, label string, title *TitleState) {
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
		}
	}
}

func refreshWorker(ctx context.Context, triggers <-chan struct{}, fetch Fetch, items []TitleSetter, title *TitleState, onFetchError func(error), fatal chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-triggers:
		}
		err := ApplyRefresh(ctx, fetch, items, title)
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
	for {
		select {
		case <-ctx.Done():
			return
		case <-quitClicked:
			systray.Quit()
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
