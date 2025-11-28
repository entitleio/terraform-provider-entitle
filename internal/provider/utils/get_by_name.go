package utils

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// NameableID represents an item that has getters for a name and an ID.
type NameableID interface {
	GetName() string
	GetID() uuid.UUID
}

// fetchPageFn fetches a page of results and returns:
//   - items: slice of T
//   - totalPages: number of pages
//   - error if any
type fetchPageFn[T NameableID] func(ctx context.Context, page int) (items []T, totalPages int, err error)

// FindIDByName finds the ID of an item by name.
func FindIDByName[T NameableID](ctx context.Context, name string, fetch fetchPageFn[T]) (*uuid.UUID, error) {
	page := 1

	for {
		items, totalPages, err := fetch(ctx, page)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			if item.GetName() == name {
				id := item.GetID()
				return &id, nil
			}
		}

		if page >= totalPages {
			break
		}
		page++
	}

	return nil, fmt.Errorf("item with name %q not found", name)
}
