package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	autoRefreshClaude := flag.Bool("claude-auto-refresh", false, "Automatically refresh stale Claude usage with claude -p /usage")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	registrations := provider.DefaultRegistrations(*autoRefreshClaude)
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
	})
}
