package provider

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"ratelimitbar/internal/claudeusage"
	"ratelimitbar/internal/usage"
)

func TestRunClaudeUsageWaitsForStatusLineAfterClosingScreen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude-usage.json")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.Command("/bin/sh", "-c", `stty -icanon -echo; dd bs=1 count=1 >/dev/null 2>&1; printf '%s' "$1" > "$2"; sleep 30`, "claude", `{"fetched_at":"`+time.Now().Add(time.Second).UTC().Format(time.RFC3339Nano)+`","rate_limits":{"five_hour":{"used_percentage":12,"resets_at":4102444800}}}`, path)
	if err := runClaudeUsage(ctx, cmd, path); err != nil {
		t.Fatal(err)
	}
	snapshot, err := claudeusage.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot.FiveHour, &claudeusage.Window{UsedPercentage: 12, ResetsAt: time.Unix(4102444800, 0)}) {
		t.Fatalf("FiveHour = %+v", snapshot.FiveHour)
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("Claude is still running: %v", err)
	}
}

func TestRunClaudeUsageTimesOutWithoutNewCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cmd := exec.Command("/bin/sh", "-c", "sleep 30")
	err := runClaudeUsage(ctx, cmd, filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("Claude is still running: %v", err)
	}
}

func TestRunClaudeUsageReportsEarlyExit(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exit 2")
	err := runClaudeUsage(context.Background(), cmd, filepath.Join(t.TempDir(), "missing.json"))
	if err == nil || !strings.Contains(err.Error(), "exit status 2") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunClaudeUsageDoesNotAcceptUnchangedCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.json")
	if err := claudeusage.Capture(strings.NewReader(`{"rate_limits":{"five_hour":{"used_percentage":12,"resets_at":4102444800}}}`), io.Discard, path, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1100*time.Millisecond)
	defer cancel()
	cmd := exec.Command("/bin/sh", "-c", "sleep 30")
	if err := runClaudeUsage(ctx, cmd, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want unchanged cache to time out", err)
	}
}

func TestClaudeFetchReadsCacheWrittenByRefresh(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.json")
	now := time.Now().UTC()
	p := claude{path: path, staleAfter: time.Minute, now: func() time.Time { return now }, refresh: func(ctx context.Context, actualPath string) error {
		if actualPath != path {
			t.Fatalf("path = %q", actualPath)
		}
		if _, ok := ctx.Deadline(); ok != true {
			t.Fatal("refresh has no deadline")
		}
		return claudeusage.Capture(strings.NewReader(`{"rate_limits":{"five_hour":{"used_percentage":12,"resets_at":4102444800}}}`), io.Discard, path, now)
	}}
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, usage.Usage{Windows: []usage.Window{{Label: WindowFiveHour, UsedPercent: 12, ResetsAt: time.Unix(4102444800, 0)}}, ReceivedAt: now, Stale: false}) {
		t.Fatalf("usage = %+v", got)
	}
}

func TestClaudeFetchKeepsCacheStaleWhenRefreshFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.json")
	now := time.Now().UTC()
	if err := claudeusage.Capture(strings.NewReader(`{"rate_limits":{"five_hour":{"used_percentage":12,"resets_at":4102444800}}}`), io.Discard, path, now); err != nil {
		t.Fatal(err)
	}
	p := claude{path: path, staleAfter: time.Minute, now: func() time.Time { return now }, refresh: func(context.Context, string) error { return context.DeadlineExceeded }}
	got, err := p.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, usage.Usage{Windows: []usage.Window{{Label: WindowFiveHour, UsedPercent: 12, ResetsAt: time.Unix(4102444800, 0)}}, ReceivedAt: now, Stale: true}) {
		t.Fatalf("usage = %+v", got)
	}
}

func TestClaudeFetchReportsRefreshFailureWithoutCache(t *testing.T) {
	p := claude{path: filepath.Join(t.TempDir(), "missing.json"), refresh: func(context.Context, string) error { return context.DeadlineExceeded }}
	_, err := p.Fetch(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v", err)
	}
}
