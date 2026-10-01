package provider

import (
	"context"
	"errors"
	"sync"

	"ratelimitbar/internal/usage"
)

// FetchAll は全 Provider を並行取得する。Result は providers の順で返し、
// 各 Provider の error は Result に保持したうえで errors.Join で束ねて返す。
func FetchAll(ctx context.Context, providers []Provider) ([]usage.Result, error) {
	results := make([]usage.Result, len(providers))
	var wg sync.WaitGroup
	for i, p := range providers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, err := p.Fetch(ctx)
			results[i] = usage.Result{Provider: p.Name(), Usage: u, Err: err}
		}()
	}
	wg.Wait()

	errs := make([]error, 0, len(results))
	for _, r := range results {
		errs = append(errs, r.Err)
	}
	return results, errors.Join(errs...)
}
