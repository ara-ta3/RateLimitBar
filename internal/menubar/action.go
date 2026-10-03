package menubar

import "context"

type Action struct {
	Title   func() string
	OnClick func(context.Context) bool
}

func runActionOnClick(ctx context.Context, clicked <-chan struct{}, onClick func(context.Context) bool, triggers chan<- struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-clicked:
			if onClick(ctx) {
				select {
				case triggers <- struct{}{}:
				default:
				}
			}
		}
	}
}
