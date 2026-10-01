package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"ratelimitbar/internal/usage"
)

// DefaultClaudeStaleAfter は、キャッシュが Claude Code の利用中にだけ更新されることを踏まえ、
// 60 秒周期の数回分で誤って古い扱いにならない長さにしている。
const DefaultClaudeStaleAfter = 15 * time.Minute

type claude struct {
	path       string
	pathErr    error
	staleAfter time.Duration
	now        func() time.Time
}

type claudeConfig struct {
	CachedUsageUtilization *struct {
		FetchedAtMs *int64 `json:"fetchedAtMs"`
		Utilization *struct {
			FiveHour *claudeWindow `json:"five_hour"`
			SevenDay *claudeWindow `json:"seven_day"`
		} `json:"utilization"`
	} `json:"cachedUsageUtilization"`
}

type claudeWindow struct {
	Utilization *float64 `json:"utilization"`
}

// NewClaude は、ホームディレクトリの .claude.json を読む Provider を返す。
// ホームディレクトリを特定できない場合は、Fetch が error を返す。
func NewClaude() Provider {
	home, err := os.UserHomeDir()
	if err != nil {
		return claude{pathErr: fmt.Errorf("resolve home directory: %w", err)}
	}
	return NewClaudeFromFile(filepath.Join(home, ".claude.json"), DefaultClaudeStaleAfter, time.Now)
}

func NewClaudeFromFile(path string, staleAfter time.Duration, now func() time.Time) Provider {
	return claude{path: path, staleAfter: staleAfter, now: now}
}

func (claude) Name() string { return "Claude" }

func (c claude) Fetch(context.Context) (usage.Usage, error) {
	if c.pathErr != nil {
		return usage.Usage{}, c.pathErr
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		return usage.Usage{}, fmt.Errorf("read claude config: %w", err)
	}
	var cfg claudeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return usage.Usage{}, fmt.Errorf("parse claude config: %w", err)
	}
	cache := cfg.CachedUsageUtilization
	if cache == nil {
		return usage.Usage{}, errors.New("cachedUsageUtilization not found")
	}
	if cache.FetchedAtMs == nil || *cache.FetchedAtMs <= 0 {
		return usage.Usage{}, errors.New("cachedUsageUtilization.fetchedAtMs is missing or invalid")
	}
	if cache.Utilization == nil {
		return usage.Usage{}, errors.New("cachedUsageUtilization.utilization not found")
	}
	fiveHour, err := claudeWindowUsage("5h", "five_hour", cache.Utilization.FiveHour)
	if err != nil {
		return usage.Usage{}, err
	}
	weekly, err := claudeWindowUsage("Weekly", "seven_day", cache.Utilization.SevenDay)
	if err != nil {
		return usage.Usage{}, err
	}
	age := c.now().Sub(time.UnixMilli(*cache.FetchedAtMs))
	return usage.Usage{
		Windows: []usage.Window{fiveHour, weekly},
		Stale:   age > c.staleAfter,
	}, nil
}

func claudeWindowUsage(label, key string, w *claudeWindow) (usage.Window, error) {
	if w == nil || w.Utilization == nil {
		return usage.Window{}, fmt.Errorf("cachedUsageUtilization.utilization.%s is missing", key)
	}
	return usage.Window{Label: label, UsedPercent: int(math.Round(*w.Utilization))}, nil
}
