package menubar

import "context"

type Option struct {
	Title    string
	OnChange func(bool)
}

type checkbox interface {
	Check()
	Uncheck()
}

func toggleOptionOnClick(ctx context.Context, clicked <-chan struct{}, item checkbox, onChange func(bool)) {
	enabled := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-clicked:
			enabled = !enabled
			onChange(enabled)
			if enabled {
				item.Check()
			} else {
				item.Uncheck()
			}
		}
	}
}
