package bulk

import (
	"context"
	"sync"
)

// ForEach runs fn for every item with at most parallelism concurrent calls.
// The first error cancels the context passed to the remaining calls and is returned.
func ForEach[T any](ctx context.Context, parallelism int, items []T, fn func(ctx context.Context, item T) error) error {
	if parallelism < 1 {
		parallelism = 1
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
		sem      = make(chan struct{}, parallelism)
	)

loop:
	for _, item := range items {
		select {
		case sem <- struct{}{}:
		case <-runCtx.Done():
			break loop
		}

		wg.Add(1)
		go func(item T) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := fn(runCtx, item); err != nil {
				once.Do(func() {
					firstErr = err
					cancel()
				})
			}
		}(item)
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	return ctx.Err()
}

// FilterIDs calls keep for every id (with bounded parallelism) and returns the
// ids for which keep returned true, in their original order. Bulk resources use
// it in Read to drop IDs that no longer exist or have drifted.
func FilterIDs(ctx context.Context, parallelism int, ids []string, keep func(ctx context.Context, id string) (bool, error)) ([]string, error) {
	results := make([]bool, len(ids))
	indexes := make([]int, len(ids))
	for i := range ids {
		indexes[i] = i
	}

	err := ForEach(ctx, parallelism, indexes, func(ctx context.Context, i int) error {
		ok, err := keep(ctx, ids[i])
		if err != nil {
			return err
		}
		results[i] = ok
		return nil
	})
	if err != nil {
		return nil, err
	}

	kept := make([]string, 0, len(ids))
	for i, ok := range results {
		if ok {
			kept = append(kept, ids[i])
		}
	}

	return kept, nil
}
