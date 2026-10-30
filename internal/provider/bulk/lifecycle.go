package bulk

import (
	"context"
	"sort"
)

// InspectFunc reads the entities in want (id -> values to check), ideally in
// a few batched requests, and reports for each one which of its values it has
// plus any other drift (Presence.Mismatch). Entities that do not exist are left
// out of the result (or reported with Exists: false).
type InspectFunc func(ctx context.Context, want map[string][]string) (map[string]Presence, error)

// PlanAdditive reads the entities the plan needs (see Additive.ToInspect) and
// computes the operations. Shared by every additive bulk resource.
func PlanAdditive(ctx context.Context, a Additive, inspect InspectFunc) (AdditivePlan, error) {
	present := map[string][]string{}

	if want := a.ToInspect(); len(want) > 0 {
		observed, err := inspect(ctx, want)
		if err != nil {
			return AdditivePlan{}, err
		}
		for id, p := range observed {
			present[id] = p.Values
		}
	}

	return a.Plan(present), nil
}

// Refresh is the shared Read step of bulk resources: it inspects every managed
// entity (state IDs plus drifted ones) and returns the IDs that are fully in
// sync, together with the updated ownership to store in private state.
func Refresh(ctx context.Context, stateIDs []string, ownership Ownership, values []string, inspect InspectFunc) ([]string, Ownership, error) {
	managed := ownership.Managed(stateIDs)

	observed := map[string]Presence{}
	if len(managed) > 0 {
		want := make(map[string][]string, len(managed))
		for _, id := range managed {
			want[id] = values
		}
		var err error
		if observed, err = inspect(ctx, want); err != nil {
			return nil, Ownership{}, err
		}
	}

	inSync, next := ownership.Reconcile(managed, observed, values)
	return inSync, next, nil
}

// SortedKeys returns the keys of m in sorted order.
func SortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Group is a set of entities that receive an identical update.
type Group[T any] struct {
	IDs   []string
	Value T
}

// GroupBy merges entities whose values have the same key, so each distinct
// update becomes one bulk operation. Entities whose key is "" are skipped
// (nothing to send). Output is deterministic.
func GroupBy[T any](items map[string]T, key func(T) string) []Group[T] {
	byKey := map[string]*Group[T]{}
	for id, v := range items {
		k := key(v)
		if k == "" {
			continue
		}
		g, ok := byKey[k]
		if !ok {
			g = &Group[T]{Value: v}
			byKey[k] = g
		}
		g.IDs = append(g.IDs, id)
	}

	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	groups := make([]Group[T], 0, len(keys))
	for _, k := range keys {
		g := byKey[k]
		sort.Strings(g.IDs)
		groups = append(groups, *g)
	}
	return groups
}
