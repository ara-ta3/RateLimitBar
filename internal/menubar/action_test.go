package menubar

import (
	"context"
	"testing"
	"time"
)

func TestActionTriggersRefreshAfterSettingsChange(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clicked := make(chan struct{}, 1)
	triggers := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runActionOnClick(ctx, clicked, func(context.Context) bool { return true }, triggers)
	}()
	clicked <- struct{}{}
	select {
	case <-triggers:
	case <-time.After(time.Second):
		t.Fatal("settings change did not trigger refresh")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("action worker did not stop")
	}
}

func TestCanceledSelectionDoesNotTriggerRefresh(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	clicked := make(chan struct{}, 1)
	triggers := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runActionOnClick(ctx, clicked, func(context.Context) bool {
			cancel()
			return false
		}, triggers)
	}()
	clicked <- struct{}{}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("action worker did not stop")
	}
	if got := len(triggers); got != 0 {
		t.Fatalf("pending refreshes = %d, want 0 after cancel", got)
	}
}
