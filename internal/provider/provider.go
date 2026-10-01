package provider

import (
	"context"

	"ratelimitbar/internal/usage"
)

type Provider interface {
	Name() string
	Fetch(ctx context.Context) (usage.Usage, error)
}
