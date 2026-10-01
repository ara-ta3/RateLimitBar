package menubar

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ratelimitbar/internal/usage"
)

type titleRecorder struct {
	mu    sync.Mutex
	title string
}

func (r *titleRecorder) SetTitle(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.title = s
}

func (r *titleRecorder) Title() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.title
}

func TestPeriodicTriggerSendsRepeatedlyAndStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	triggers := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		periodicTrigger(ctx, 5*time.Millisecond, triggers)
		close(done)
	}()

	for i := 0; i < 3; i++ {
		select {
		case <-triggers:
		case <-time.After(2 * time.Second):
			t.Fatalf("trigger %d did not arrive", i+1)
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("periodicTrigger did not stop after cancel")
	}
}

func TestPeriodicTriggerDoesNotBlockWhenTriggerIsPending(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	triggers := make(chan struct{}, 1)
	triggers <- struct{}{} // 誰も受け取らない
	done := make(chan struct{})
	go func() {
		periodicTrigger(ctx, time.Millisecond, triggers)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("periodicTrigger blocked on a full triggers channel")
	}
}

func TestRefreshWorkerRecoversOnPeriodicTriggerAfterFetchFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	item := &titleRecorder{}
	var calls atomic.Int32
	fetchErrs := make(chan error, 8)
	fetch := func(context.Context) ([]usage.Result, error) {
		if calls.Add(1) == 1 {
			err := errors.New("claude.json unreadable")
			return []usage.Result{{Provider: "Claude", Err: err}}, err
		}
		return []usage.Result{{Provider: "Claude", Usage: usage.Usage{Windows: []usage.Window{{Label: "5h", UsedPercent: 23}}}}}, nil
	}
	triggers := make(chan struct{}, 1)
	triggers <- struct{}{}
	fatal := make(chan error, 1)
	go refreshWorker(ctx, triggers, fetch, []TitleSetter{item}, freshTitleState(), func(err error) { fetchErrs <- err }, fatal)
	go periodicTrigger(ctx, 5*time.Millisecond, triggers)

	deadline := time.After(2 * time.Second)
	for !strings.Contains(item.Title(), "23%") {
		select {
		case <-deadline:
			t.Fatalf("item never recovered, title = %q", item.Title())
		case <-time.After(2 * time.Millisecond):
		}
	}
	select {
	case <-fetchErrs:
	default:
		t.Error("first fetch error was not reported to onFetchError")
	}
	select {
	case err := <-fatal:
		t.Fatalf("worker reported fatal error: %v", err)
	default:
	}
}

func TestRefreshWorkerNeverRunsFetchConcurrentlyWithPeriodicAndManualTriggers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var running, maxRunning, calls atomic.Int32
	fetch := func(context.Context) ([]usage.Result, error) {
		n := running.Add(1)
		for {
			m := maxRunning.Load()
			if n <= m || maxRunning.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(3 * time.Millisecond)
		running.Add(-1)
		calls.Add(1)
		return []usage.Result{{Provider: "Claude"}}, nil
	}
	triggers := make(chan struct{}, 1)
	fatal := make(chan error, 1)
	go refreshWorker(ctx, triggers, fetch, []TitleSetter{&titleRecorder{}}, freshTitleState(), func(error) {}, fatal)
	go periodicTrigger(ctx, time.Millisecond, triggers)
	go func() { // 手動更新相当
		for ctx.Err() == nil {
			select {
			case triggers <- struct{}{}:
			default:
			}
			time.Sleep(time.Millisecond)
		}
	}()

	deadline := time.After(2 * time.Second)
	for calls.Load() < 5 {
		select {
		case <-deadline:
			t.Fatalf("fetch ran only %d times", calls.Load())
		case <-time.After(2 * time.Millisecond):
		}
	}
	if got := maxRunning.Load(); got != 1 {
		t.Errorf("max concurrent fetches = %d, want 1", got)
	}
}

func freshTitleState() *TitleState {
	return NewTitleState(&titleRecorder{}, NewSelection([]Source{{Name: "Claude", Windows: []string{"5h"}}}))
}
