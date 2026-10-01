package menubar

import (
	"context"
	"errors"

	"fyne.io/systray"
)

const loadingTitle = "Loading..."

// Run はメニューバーに常駐し、Quit が選ばれるまでブロックする。
// 取得は別の goroutine で行い、イベントループをブロックしない。
// 取得 error は表示に反映したうえで onFetchError に渡して継続する。
// 表示を更新できない error は、終了して呼び出し元へ返す。
func Run(providerNames []string, fetch Fetch, onFetchError func(error)) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fatal := make(chan error, 1)
	systray.Run(func() {
		systray.SetTitle("RateLimit")
		setters := make([]TitleSetter, len(providerNames))
		for i, name := range providerNames {
			item := systray.AddMenuItem(name+": "+loadingTitle, "")
			item.Disable()
			setters[i] = item
		}
		systray.AddSeparator()
		refresh := systray.AddMenuItem("Refresh", "")
		quit := systray.AddMenuItem("Quit", "")

		triggers := make(chan struct{}, 1)
		triggers <- struct{}{}
		go refreshWorker(ctx, triggers, fetch, setters, onFetchError, fatal)
		go dispatchEvents(ctx, refresh.ClickedCh, quit.ClickedCh, triggers)
	}, func() {})

	select {
	case err := <-fatal:
		return err
	default:
		return nil
	}
}

func refreshWorker(ctx context.Context, triggers <-chan struct{}, fetch Fetch, items []TitleSetter, onFetchError func(error), fatal chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-triggers:
		}
		err := ApplyRefresh(ctx, fetch, items)
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
