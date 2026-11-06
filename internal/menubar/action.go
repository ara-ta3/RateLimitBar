package menubar

import (
	"context"

	"ratelimitbar/internal/popover"
)

type Action struct {
	Kind    popover.CommandKind
	Checked func() bool
	Title   func() string
	Label   string
	Detail  func() string
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
