package menubar

import (
	"context"
	"testing"
	"time"
)

type optionCheckbox struct{ changes chan bool }

func (c optionCheckbox) Check()   { c.changes <- true }
func (c optionCheckbox) Uncheck() { c.changes <- false }

func TestMenuOptionEnablesAndDisablesAutoRefresh(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clicked := make(chan struct{})
	settingChanges := make(chan bool, 2)
	checkboxChanges := make(chan bool, 2)
	done := make(chan struct{})
	go func() {
		defer close(done)
		toggleOptionOnClick(ctx, clicked, optionCheckbox{checkboxChanges}, func(enabled bool) { settingChanges <- enabled })
	}()
	clicked <- struct{}{}
	if enabled := <-settingChanges; enabled != true {
		t.Fatalf("first setting = %v, want true", enabled)
	}
	if checked := <-checkboxChanges; checked != true {
		t.Fatalf("first checkbox = %v, want true", checked)
	}
	clicked <- struct{}{}
	if enabled := <-settingChanges; enabled != false {
		t.Fatalf("second setting = %v, want false", enabled)
	}
	if checked := <-checkboxChanges; checked != false {
		t.Fatalf("second checkbox = %v, want false", checked)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("option worker did not stop")
	}
}
