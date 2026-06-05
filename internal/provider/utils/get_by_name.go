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

	var id *uuid.UUID
	for {
		items, totalPages, err := fetch(ctx, page)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			if item.GetName() == name {
				if id != nil {
					return nil, fmt.Errorf("found multiple IDs for %s", name)
				}

				itemID := item.GetID()
				id = &itemID
			}
		}

		if page >= totalPages {
			break
		}
		page++
	}

	if id != nil {
		return id, nil
	}

	return nil, fmt.Errorf("item with name %q not found", name)
}
