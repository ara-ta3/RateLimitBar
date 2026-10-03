package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"ratelimitbar/internal/config"
	"ratelimitbar/internal/menubar"
	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

const refreshInterval = 60 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ratelimitbar:", err)
		os.Exit(1)
	}
}

func run() error {
	var autoRefreshClaude atomic.Bool

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	settings, err := config.NewStore(filepath.Join(configDir, "RateLimitBar", "settings.json"))
	if err != nil {
		reportSettingError(ctx, fmt.Errorf("設定を読み込めませんでした。取得元を再指定してください: %w", err))
	}
	registrations := provider.DefaultRegistrations(provider.Config{
		AutoRefreshClaude: autoRefreshClaude.Load,
		CodexPath:         func() string { return settings.Settings().CodexPath },
		ClaudePath:        func() string { return settings.Settings().ClaudePath },
	})
	providers := make([]provider.Provider, len(registrations))
	sources := make([]menubar.Source, len(registrations))
	for i, r := range registrations {
		providers[i] = r.Provider
		sources[i] = menubar.Source{Name: r.Provider.Name(), Windows: r.Windows}
	}
	fetch := func(ctx context.Context) ([]usage.Result, error) {
		return provider.FetchAll(ctx, providers)
	}
	return menubar.Run(ctx, sources, refreshInterval, fetch, func(err error) {
		log.Printf("fetch failed: %v", err)
	}, sourceActions(settings), menubar.Option{Title: "Claudeのキャッシュを自動更新", OnChange: autoRefreshClaude.Store})
}
