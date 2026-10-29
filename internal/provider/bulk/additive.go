package bulk

// Shared planning for "additive" bulk resources: resources that add a set of
// values (e.g. tags) to many entities and must, on removal, take away only the
// values they added themselves.
//
// Ownership is kept in provider-private state:
//   - Preexisting: per entity, managed values that were already present before
//     this resource added them. These are never removed.
//   - Drifted: entities that still exist but lost a managed value outside
//     Terraform. Read drops them from the user-visible ID set (so the plan shows
//     a change), but they remain managed so their ownership is not forgotten.

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

const ownershipPrivateKey = "bulk_ownership"

// PrivateGetter is satisfied by the framework's private state on requests.
type PrivateGetter interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

// PrivateSetter is satisfied by the framework's private state on responses.
type PrivateSetter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// Ownership records which managed values the resource does not own.
type Ownership struct {
	Preexisting map[string][]string `json:"preexisting,omitempty"`
	Drifted     []string            `json:"drifted,omitempty"`
}

// LoadOwnership reads ownership from private state. Missing data yields an empty Ownership.
func LoadOwnership(ctx context.Context, p PrivateGetter) (Ownership, diag.Diagnostics) {
	var o Ownership
	raw, diags := p.GetKey(ctx, ownershipPrivateKey)
	if diags.HasError() || len(raw) == 0 {
		return o, diags
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		diags.AddError("Invalid Private State", fmt.Sprintf("Unable to decode bulk ownership data: %s", err))
	}
	return o, diags
}

// Save writes ownership to private state in a deterministic form.
func (o Ownership) Save(ctx context.Context, p PrivateSetter) diag.Diagnostics {
	norm := Ownership{Drifted: sortedUnique(o.Drifted)}
	for id, values := range o.Preexisting {
		if len(values) > 0 {
			if norm.Preexisting == nil {
				norm.Preexisting = map[string][]string{}
			}
			norm.Preexisting[id] = sortedUnique(values)
		}
	}

	raw, err := json.Marshal(norm)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Invalid Private State", fmt.Sprintf("Unable to encode bulk ownership data: %s", err))
		return diags
	}
	return p.SetKey(ctx, ownershipPrivateKey, raw)
}

// Managed returns every entity the resource currently manages: the IDs in state plus drifted ones.
func (o Ownership) Managed(stateIDs []string) []string {
	return Unique(append(slices.Clone(stateIDs), o.Drifted...))
}

// Owned returns the managed values the resource owns (may remove) on entity id.
func (o Ownership) Owned(id string, values []string) []string {
	return Difference(values, o.Preexisting[id])
}

// Presence describes an entity as observed in the API.
type Presence struct {
	Exists bool
	// Values lists which of the requested values the entity currently has.
	Values []string
	// Mismatch reports drift in anything other than the additive values
	// (e.g. a setting that differs from configuration).
	Mismatch bool
}

// Inspect runs fn for every id with bounded parallelism and collects the results.
func Inspect(ctx context.Context, parallelism int, ids []string, fn func(ctx context.Context, id string) (Presence, error)) (map[string]Presence, error) {
	var mu sync.Mutex
	out := make(map[string]Presence, len(ids))

	err := ForEach(ctx, parallelism, ids, func(ctx context.Context, id string) error {
		p, err := fn(ctx, id)
		if err != nil {
			return err
		}
		mu.Lock()
		out[id] = p
		mu.Unlock()
		return nil
	})

	return out, err
}

// Reconcile classifies the managed entities after Read. It returns the IDs that
// carry every value and have no other mismatch (for state) and the updated
// ownership: entities that no longer exist are forgotten, entities missing a
// value or reporting a mismatch become drifted, and a
// pre-existing value that disappeared is no longer considered pre-existing (if
// re-added, the resource now owns it).
func (o Ownership) Reconcile(managed []string, observed map[string]Presence, values []string) ([]string, Ownership) {
	next := Ownership{Preexisting: map[string][]string{}}
	var inSync []string

	for _, id := range managed {
		p := observed[id]
		if !p.Exists {
			continue
		}

		if pre := Intersection(o.Preexisting[id], p.Values); len(pre) > 0 {
			next.Preexisting[id] = pre
		}

		if !p.Mismatch && ContainsAll(p.Values, values) {
			inSync = append(inSync, id)
		} else {
			next.Drifted = append(next.Drifted, id)
		}
	}

	return inSync, next
}

// Change is the set of values to add to and remove from one entity.
type Change struct {
	Add    []string
	Remove []string
}

// IsEmpty reports whether the change does nothing.
func (c Change) IsEmpty() bool { return len(c.Add) == 0 && len(c.Remove) == 0 }

// ChangeGroup is a Change shared by several entities, i.e. one bulk operation.
type ChangeGroup struct {
	IDs []string
	Change
}

// GroupChanges merges entities with identical changes so each distinct change
// becomes one bulk operation. Empty changes are dropped. Output is deterministic.
func GroupChanges(changes map[string]Change) []ChangeGroup {
	byKey := map[string]*ChangeGroup{}
	for id, c := range changes {
		if c.IsEmpty() {
			continue
		}
		add, remove := sortedUnique(c.Add), sortedUnique(c.Remove)
		key := strings.Join(add, "\x00") + "\x01" + strings.Join(remove, "\x00")
		g, ok := byKey[key]
		if !ok {
			g = &ChangeGroup{Change: Change{Add: add, Remove: remove}}
			byKey[key] = g
		}
		g.IDs = append(g.IDs, id)
	}

	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	groups := make([]ChangeGroup, 0, len(keys))
	for _, k := range keys {
		g := byKey[k]
		sort.Strings(g.IDs)
		groups = append(groups, *g)
	}
	return groups
}

// Additive plans a transition of an additive resource from its prior state to its plan.
type Additive struct {
	OldIDs    []string // IDs in prior state (empty on create)
	NewIDs    []string // IDs in plan
	OldValues []string // values in prior state (empty on create)
	NewValues []string // values in plan
	Prior     Ownership
}

func (a Additive) ids() SetDiff    { return Diff(a.Prior.Managed(a.OldIDs), a.NewIDs) }
func (a Additive) values() SetDiff { return Diff(a.OldValues, a.NewValues) }

// IDs returns how the managed entities change: drifted entities count as
// previously managed, so a drifted entity still in the plan is Kept.
func (a Additive) IDs() SetDiff { return a.ids() }

// IsDrifted reports whether id was recorded as drifted by the last Read.
func (a Additive) IsDrifted(id string) bool { return slices.Contains(a.Prior.Drifted, id) }

// ToInspect returns, per entity, the values whose current presence must be
// read before writing, so values that already exist are recorded as
// pre-existing instead of owned.
func (a Additive) ToInspect() map[string][]string {
	ids, values := a.ids(), a.values()
	out := map[string][]string{}

	if len(a.NewValues) == 0 {
		return out // nothing to check (e.g. a settings resource without additive values)
	}
	for _, id := range ids.Added {
		out[id] = a.NewValues
	}
	if len(values.Added) > 0 {
		for _, id := range ids.Kept {
			out[id] = values.Added
		}
	}
	return out
}

// AdditivePlan is the outcome of planning.
type AdditivePlan struct {
	// Removals apply to entities no longer managed; missing entities can be skipped.
	Removals []ChangeGroup
	// Changes apply to entities that stay or become managed.
	Changes []ChangeGroup
	// RemovalsByID and ChangesByID hold the same changes per entity, for
	// resources that merge them with other updates before grouping.
	RemovalsByID map[string]Change
	ChangesByID  map[string]Change
	// Next is the ownership to store when every operation succeeds.
	Next Ownership
	// OnError is a conservative ownership to store if applying fails part-way:
	// it remembers everything learned about pre-existing values so a retry
	// never mistakes partially added values for pre-existing ones.
	OnError Ownership
}

// Plan computes the operations, using present (the values found by inspecting
// ToInspect) to decide which values are pre-existing.
func (a Additive) Plan(present map[string][]string) AdditivePlan {
	ids, values := a.ids(), a.values()
	drifted := toSet(a.Prior.Drifted)

	removals := map[string]Change{}
	changes := map[string]Change{}
	next := Ownership{Preexisting: map[string][]string{}}

	for _, id := range ids.Removed {
		removals[id] = Change{Remove: a.Prior.Owned(id, a.OldValues)}
	}

	for _, id := range ids.Kept {
		pre := Unique(append(Intersection(a.Prior.Preexisting[id], a.NewValues), present[id]...))
		add := values.Added
		if _, ok := drifted[id]; ok {
			add = a.NewValues // re-add whatever went missing
		}
		changes[id] = Change{
			Add:    Difference(add, pre),
			Remove: a.Prior.Owned(id, values.Removed),
		}
		if len(pre) > 0 {
			next.Preexisting[id] = pre
		}
	}

	for _, id := range ids.Added {
		pre := present[id]
		add := Difference(a.NewValues, pre)
		if len(add) == 0 {
			// Still send a request so a non-existent entity is reported by the API.
			add = a.NewValues
		}
		changes[id] = Change{Add: add}
		if len(pre) > 0 {
			next.Preexisting[id] = pre
		}
	}

	onError := Ownership{Preexisting: map[string][]string{}, Drifted: a.Prior.Drifted}
	for _, id := range Unique(append(a.Prior.Managed(a.OldIDs), a.NewIDs...)) {
		if pre := Unique(append(slices.Clone(a.Prior.Preexisting[id]), present[id]...)); len(pre) > 0 {
			onError.Preexisting[id] = pre
		}
	}

	return AdditivePlan{
		Removals:     GroupChanges(removals),
		Changes:      GroupChanges(changes),
		RemovalsByID: removals,
		ChangesByID:  changes,
		Next:         next,
		OnError:      onError,
	}
}

// RemoveAll plans removing every owned value from every managed entity (destroy).
func (o Ownership) RemoveAll(stateIDs, values []string) []ChangeGroup {
	changes := map[string]Change{}
	for _, id := range o.Managed(stateIDs) {
		changes[id] = Change{Remove: o.Owned(id, values)}
	}
	return GroupChanges(changes)
}

func sortedUnique(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := Unique(items)
	sort.Strings(out)
	return out
}
