package provider_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

type fakeProvider struct {
	name  string
	fetch func(ctx context.Context) (usage.Usage, error)
}

func (f fakeProvider) Name() string { return f.name }
func (f fakeProvider) Fetch(ctx context.Context) (usage.Usage, error) {
	return f.fetch(ctx)
}

func fixedUsage(label string, percent int) func(context.Context) (usage.Usage, error) {
	return func(context.Context) (usage.Usage, error) {
		return usage.Usage{Windows: []usage.Window{{Label: label, UsedPercent: percent}}}, nil
	}
}

func TestFetchAllKeepsOtherResultsWhenOneProviderFails(t *testing.T) {
	codexErr := errors.New("codex unavailable")
	providers := []provider.Provider{
		fakeProvider{"Claude", fixedUsage("5h", 23)},
		fakeProvider{"Codex", func(context.Context) (usage.Usage, error) { return usage.Usage{}, codexErr }},
		fakeProvider{"Cursor", fixedUsage("Monthly", 18)},
	}

	results, err := provider.FetchAll(context.Background(), providers)

	if !errors.Is(err, codexErr) {
		t.Fatalf("returned error = %v, want it to wrap %v", err, codexErr)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	for _, i := range []int{0, 2} {
		if results[i].Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, results[i].Err)
		}
		if len(results[i].Usage.Windows) != 1 {
			t.Errorf("results[%d].Usage.Windows = %v, want 1 window", i, results[i].Usage.Windows)
		}
	}
	if got := results[0].Usage.Windows[0]; got != (usage.Window{Label: "5h", UsedPercent: 23}) {
		t.Errorf("Claude window = %v", got)
	}
	if got := results[2].Usage.Windows[0]; got != (usage.Window{Label: "Monthly", UsedPercent: 18}) {
		t.Errorf("Cursor window = %v", got)
	}
	if !errors.Is(results[1].Err, codexErr) {
		t.Errorf("results[1].Err = %v, want %v", results[1].Err, codexErr)
	}
}

func TestFetchAllJoinsEveryProviderError(t *testing.T) {
	errA, errB := errors.New("a failed"), errors.New("b failed")
	providers := []provider.Provider{
		fakeProvider{"A", func(context.Context) (usage.Usage, error) { return usage.Usage{}, errA }},
		fakeProvider{"B", func(context.Context) (usage.Usage, error) { return usage.Usage{}, errB }},
	}

	_, err := provider.FetchAll(context.Background(), providers)

	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatalf("returned error = %v, want it to wrap both %v and %v", err, errA, errB)
	}
}

func TestFetchAllReturnsNoErrorWhenAllSucceed(t *testing.T) {
	results, err := provider.FetchAll(context.Background(), []provider.Provider{
		provider.NewMockClaude(), provider.NewMockCodex(), provider.NewMockCursor(),
	})
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
}

func TestFetchAllPreservesRegistrationOrder(t *testing.T) {
	names := []string{"Claude", "Codex", "Cursor"}
	providers := make([]provider.Provider, len(names))
	for i, n := range names {
		providers[i] = fakeProvider{n, fixedUsage("5h", i)}
	}

	results, err := provider.FetchAll(context.Background(), providers)
	if err != nil {
		t.Fatal(err)
	}
	for i, n := range names {
		if results[i].Provider != n {
			t.Errorf("results[%d].Provider = %q, want %q", i, results[i].Provider, n)
		}
	}
}

func TestFetchAllFetchesProvidersConcurrently(t *testing.T) {
	const n = 3
	var mu sync.Mutex
	started := 0
	allStarted := make(chan struct{})
	barrier := func(ctx context.Context) (usage.Usage, error) {
		mu.Lock()
		started++
		if started == n {
			close(allStarted)
		}
		mu.Unlock()
		select {
		case <-allStarted:
			return usage.Usage{}, nil
		case <-ctx.Done():
			return usage.Usage{}, ctx.Err()
		}
	}
	providers := []provider.Provider{
		fakeProvider{"A", barrier}, fakeProvider{"B", barrier}, fakeProvider{"C", barrier},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := provider.FetchAll(ctx, providers); err != nil {
		t.Fatalf("providers were not fetched concurrently: %v", err)
	}
}
