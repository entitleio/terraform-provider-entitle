package utils

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ExternalIDGetter represents an item that has getters for an external ID and an ID.
type ExternalIDGetter interface {
	GetExternalID() string
	GetID() uuid.UUID
}

// fetchEIDPageFn fetches a page of results and returns:
//   - items: slice of T
//   - totalPages: number of pages
//   - error if any
type fetchEIDPageFn[T ExternalIDGetter] func(ctx context.Context, page int) (items []T, totalPages int, err error)

// FindIDByExternalID finds the ID of an item by external id.
func FindIDByExternalID[T ExternalIDGetter](ctx context.Context, externalID string, fetch fetchEIDPageFn[T]) (*uuid.UUID, error) {
	page := 1

	for {
		items, totalPages, err := fetch(ctx, page)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			if item.GetExternalID() == externalID {
				return new(item.GetID()), nil
			}
		}

		if page >= totalPages {
			break
		}
		page++
	}

	return nil, fmt.Errorf("item with external ID %q not found", externalID)
}
