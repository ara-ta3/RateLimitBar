package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"ratelimitbar/internal/menubar"
	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ratelimitbar:", err)
		os.Exit(1)
	}
}

func run() error {
	providers := []provider.Provider{
		provider.NewMockClaude(),
		provider.NewMockCodex(),
		provider.NewMockCursor(),
	}
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	fetch := func(ctx context.Context) ([]usage.Result, error) {
		return provider.FetchAll(ctx, providers)
	}
	return menubar.Run(names, fetch, func(err error) {
		log.Printf("fetch failed: %v", err)
	})
}
