package provider

import (
	"context"
	"errors"
	"math"
	"time"

	"ratelimitbar/internal/claudeusage"
	"ratelimitbar/internal/usage"
)

const DefaultClaudeStaleAfter = 15 * time.Minute

type claude struct {
	path       string
	pathErr    error
	staleAfter time.Duration
	now        func() time.Time
}

func claudeWindowLabels() []string { return []string{WindowFiveHour, WindowWeekly} }

func NewClaude() Provider {
	path, err := claudeusage.CachePath()
	return claude{path: path, pathErr: err, staleAfter: DefaultClaudeStaleAfter, now: time.Now}
}

func NewClaudeFromFile(path string, staleAfter time.Duration, now func() time.Time) Provider {
	return claude{path: path, staleAfter: staleAfter, now: now}
}

func (claude) Name() string { return "Claude" }

func (c claude) Fetch(_ context.Context) (usage.Usage, error) {
	if c.pathErr != nil {
		return usage.Usage{}, c.pathErr
	}
	snapshot, err := claudeusage.Read(c.path)
	if err != nil {
		return usage.Usage{}, err
	}
	now := c.now()
	windows := make([]usage.Window, 0, 2)
	if w := snapshot.FiveHour; w != nil && now.Before(w.ResetsAt) {
		windows = append(windows, usage.Window{Label: WindowFiveHour, UsedPercent: int(math.Round(w.UsedPercentage)), ResetsAt: w.ResetsAt})
	}
	if w := snapshot.SevenDay; w != nil && now.Before(w.ResetsAt) {
		windows = append(windows, usage.Window{Label: WindowWeekly, UsedPercent: int(math.Round(w.UsedPercentage)), ResetsAt: w.ResetsAt})
	}
	if len(windows) == 0 {
		return usage.Usage{}, errors.New("Claude statusLine cache has no active rate limit windows; use Claude Code to update it")
	}
	return usage.Usage{Windows: windows, ReceivedAt: snapshot.FetchedAt, Stale: now.Sub(snapshot.FetchedAt) > c.staleAfter}, nil
}
