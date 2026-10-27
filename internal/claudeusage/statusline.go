package claudeusage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Window struct {
	UsedPercentage float64
	ResetsAt       time.Time
}

type Snapshot struct {
	FetchedAt time.Time
	FiveHour  *Window
	SevenDay  *Window
}

type windowJSON struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       *int64   `json:"resets_at"`
}

type rateLimitsJSON struct {
	FiveHour *windowJSON `json:"five_hour,omitempty"`
	SevenDay *windowJSON `json:"seven_day,omitempty"`
}

type payload struct {
	FetchedAt  time.Time       `json:"fetched_at"`
	RateLimits *rateLimitsJSON `json:"rate_limits"`
}

func CachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "RateLimitBar", "claude-usage.json"), nil
}

func parseWindow(w *windowJSON) (*Window, error) {
	if w == nil {
		return nil, nil
	}
	if w.UsedPercentage == nil || *w.UsedPercentage < 0 || *w.UsedPercentage > 100 {
		return nil, errors.New("used_percentage must be a number from 0 to 100")
	}
	if w.ResetsAt == nil || *w.ResetsAt <= 0 {
		return nil, errors.New("resets_at must be a positive Unix timestamp")
	}
	return &Window{UsedPercentage: *w.UsedPercentage, ResetsAt: time.Unix(*w.ResetsAt, 0)}, nil
}

func (p payload) snapshot() (Snapshot, error) {
	if p.FetchedAt.IsZero() || p.RateLimits == nil {
		return Snapshot{}, errors.New("Claude usage cache requires fetched_at and rate_limits")
	}
	fiveHour, err := parseWindow(p.RateLimits.FiveHour)
	if err != nil {
		return Snapshot{}, fmt.Errorf("five_hour: %w", err)
	}
	sevenDay, err := parseWindow(p.RateLimits.SevenDay)
	if err != nil {
		return Snapshot{}, fmt.Errorf("seven_day: %w", err)
	}
	return Snapshot{FetchedAt: p.FetchedAt, FiveHour: fiveHour, SevenDay: sevenDay}, nil
}

func Read(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read Claude statusLine cache (configure claude-statusline): %w", err)
	}
	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return Snapshot{}, fmt.Errorf("parse Claude statusLine cache: %w", err)
	}
	return p.snapshot()
}

func Capture(input io.Reader, output io.Writer, path string, now time.Time) error {
	data, err := io.ReadAll(input)
	if err != nil {
		return err
	}
	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("parse Claude statusLine input: %w", err)
	}
	if p.RateLimits == nil {
		return nil
	}
	p.FetchedAt = now
	snapshot, err := p.snapshot()
	if err != nil {
		return err
	}
	data, err = json.Marshal(p)
	if err != nil {
		return err
	}
	if err := writeCache(path, data); err != nil {
		return err
	}
	var parts []string
	if w := snapshot.FiveHour; w != nil && now.Before(w.ResetsAt) {
		parts = append(parts, fmt.Sprintf("5h %d%%", int(math.Round(w.UsedPercentage))))
	}
	if w := snapshot.SevenDay; w != nil && now.Before(w.ResetsAt) {
		parts = append(parts, fmt.Sprintf("Weekly %d%%", int(math.Round(w.UsedPercentage))))
	}
	if len(parts) > 0 {
		_, err = fmt.Fprintln(output, "Claude "+strings.Join(parts, " / "))
	}
	return err
}

func writeCache(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "claude-usage-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
