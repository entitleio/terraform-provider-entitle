package bulk

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// MaxIDsPerList is the maximum number of IDs the list endpoints accept in one request.
const MaxIDsPerList = 100

// ListByIDsFunc lists the entities with the given IDs (at most MaxIDsPerList) in one request.
type ListByIDsFunc[T any] func(ctx context.Context, ids []string) ([]T, error)

// FetchByIDs reads entities through a list endpoint, MaxIDsPerList IDs per
// request and up to parallelism requests at a time. The result is keyed by
// lower-cased ID; entities that do not exist are absent.
func FetchByIDs[T any](ctx context.Context, parallelism int, ids []string, list ListByIDsFunc[T], idOf func(T) string) (map[string]T, error) {
	var (
		mu  sync.Mutex
		out = make(map[string]T, len(ids))
	)

	err := ForEach(ctx, parallelism, Chunk(Unique(ids), MaxIDsPerList), func(ctx context.Context, chunk []string) error {
		items, err := list(ctx, chunk)
		if err != nil {
			return err
		}
		if len(items) > len(chunk) {
			// Defensive: the ids filter must not return unrelated entities.
			return fmt.Errorf("got %d results for %d ids", len(items), len(chunk))
		}

		mu.Lock()
		defer mu.Unlock()
		for _, it := range items {
			out[strings.ToLower(idOf(it))] = it
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
