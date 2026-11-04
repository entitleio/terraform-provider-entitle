package bulk

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestGroupChanges(t *testing.T) {
	got := GroupChanges(map[string]Change{
		"b": {Add: []string{"y", "x"}},
		"a": {Add: []string{"x", "y"}},
		"c": {Remove: []string{"x"}},
		"d": {}, // empty, dropped
	})
	want := []ChangeGroup{
		{IDs: []string{"c"}, Change: Change{Remove: []string{"x"}}},
		{IDs: []string{"a", "b"}, Change: Change{Add: []string{"x", "y"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestAdditiveCreate(t *testing.T) {
	a := Additive{NewIDs: []string{"r1", "r2"}, NewValues: []string{"pci", "sox"}}

	if got, want := a.ToInspect(), map[string][]string{"r1": {"pci", "sox"}, "r2": {"pci", "sox"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ToInspect = %v, want %v", got, want)
	}

	p := a.Plan(map[string][]string{"r1": {"pci"}})
	if len(p.Removals) != 0 {
		t.Fatalf("unexpected removals %+v", p.Removals)
	}
	wantChanges := []ChangeGroup{
		{IDs: []string{"r2"}, Change: Change{Add: []string{"pci", "sox"}}},
		{IDs: []string{"r1"}, Change: Change{Add: []string{"sox"}}},
	}
	if !reflect.DeepEqual(p.Changes, wantChanges) {
		t.Fatalf("changes = %+v, want %+v", p.Changes, wantChanges)
	}
	if !reflect.DeepEqual(p.Next.Preexisting, map[string][]string{"r1": {"pci"}}) {
		t.Fatalf("preexisting = %v", p.Next.Preexisting)
	}
}

func TestAdditiveCreateAllPreexistingStillSendsRequest(t *testing.T) {
	p := Additive{NewIDs: []string{"r1"}, NewValues: []string{"pci"}}.Plan(map[string][]string{"r1": {"pci"}})
	want := []ChangeGroup{{IDs: []string{"r1"}, Change: Change{Add: []string{"pci"}}}}
	if !reflect.DeepEqual(p.Changes, want) {
		t.Fatalf("changes = %+v, want %+v", p.Changes, want)
	}
}

func TestAdditiveUpdate(t *testing.T) {
	a := Additive{
		OldIDs:    []string{"r1", "r2"},
		NewIDs:    []string{"r2", "r3"},
		OldValues: []string{"pci", "sox"},
		NewValues: []string{"pci", "hipaa"},
		Prior: Ownership{Preexisting: map[string][]string{
			"r1": {"pci"},
			"r2": {"sox"},
		}},
	}

	wantInspect := map[string][]string{"r2": {"hipaa"}, "r3": {"pci", "hipaa"}}
	if got := a.ToInspect(); !reflect.DeepEqual(got, wantInspect) {
		t.Fatalf("ToInspect = %v, want %v", got, wantInspect)
	}

	p := a.Plan(map[string][]string{"r3": {"hipaa"}})

	// r1 leaves: only the owned "sox" is removed.
	wantRemovals := []ChangeGroup{{IDs: []string{"r1"}, Change: Change{Remove: []string{"sox"}}}}
	if !reflect.DeepEqual(p.Removals, wantRemovals) {
		t.Fatalf("removals = %+v, want %+v", p.Removals, wantRemovals)
	}

	// r2 stays: add "hipaa", pre-existing "sox" is not removed (no change at all for removal).
	// r3 joins: "hipaa" pre-existing, add only "pci".
	wantChanges := []ChangeGroup{
		{IDs: []string{"r2"}, Change: Change{Add: []string{"hipaa"}}},
		{IDs: []string{"r3"}, Change: Change{Add: []string{"pci"}}},
	}
	if !reflect.DeepEqual(p.Changes, wantChanges) {
		t.Fatalf("changes = %+v, want %+v", p.Changes, wantChanges)
	}

	// "sox" is no longer managed, so r2's entry is dropped; r1 is no longer managed.
	if want := map[string][]string{"r3": {"hipaa"}}; !reflect.DeepEqual(p.Next.Preexisting, want) {
		t.Fatalf("next preexisting = %v, want %v", p.Next.Preexisting, want)
	}

	// On error everything learned is kept.
	wantOnError := map[string][]string{"r1": {"pci"}, "r2": {"sox"}, "r3": {"hipaa"}}
	if !reflect.DeepEqual(p.OnError.Preexisting, wantOnError) {
		t.Fatalf("on-error preexisting = %v, want %v", p.OnError.Preexisting, wantOnError)
	}
}

func TestAdditiveDriftedIDs(t *testing.T) {
	// r1 drifted and is still configured; r2 drifted and was removed from config.
	a := Additive{
		OldIDs:    []string{"r3"},
		NewIDs:    []string{"r1", "r3"},
		OldValues: []string{"pci", "sox"},
		NewValues: []string{"pci", "sox"},
		Prior: Ownership{
			Preexisting: map[string][]string{"r1": {"pci"}, "r2": {"pci"}},
			Drifted:     []string{"r1", "r2"},
		},
	}

	if got := a.ToInspect(); len(got) != 0 {
		t.Fatalf("drifted IDs must not be re-inspected (ownership is known), got %v", got)
	}

	p := a.Plan(nil)
	wantRemovals := []ChangeGroup{{IDs: []string{"r2"}, Change: Change{Remove: []string{"sox"}}}}
	if !reflect.DeepEqual(p.Removals, wantRemovals) {
		t.Fatalf("removals = %+v, want %+v", p.Removals, wantRemovals)
	}
	wantChanges := []ChangeGroup{{IDs: []string{"r1"}, Change: Change{Add: []string{"sox"}}}}
	if !reflect.DeepEqual(p.Changes, wantChanges) {
		t.Fatalf("changes = %+v, want %+v", p.Changes, wantChanges)
	}
	if len(p.Next.Drifted) != 0 {
		t.Fatalf("drift should be resolved, got %v", p.Next.Drifted)
	}
}

func TestReconcile(t *testing.T) {
	o := Ownership{Preexisting: map[string][]string{"r1": {"pci"}, "r2": {"pci"}, "r3": {"pci"}}}
	managed := []string{"r1", "r2", "r3", "r4"}
	observed := map[string]Presence{
		"r1": {Exists: true, Values: []string{"pci", "sox"}}, // in sync
		"r2": {Exists: true, Values: []string{"sox"}},        // drifted, lost pre-existing "pci"
		"r3": {Exists: false},                                // deleted
		"r4": {Exists: true, Values: []string{"pci"}},        // drifted
	}

	inSync, next := o.Reconcile(managed, observed, []string{"pci", "sox"})

	if !reflect.DeepEqual(inSync, []string{"r1"}) {
		t.Fatalf("in sync = %v", inSync)
	}
	if !reflect.DeepEqual(next.Drifted, []string{"r2", "r4"}) {
		t.Fatalf("drifted = %v", next.Drifted)
	}
	if want := map[string][]string{"r1": {"pci"}}; !reflect.DeepEqual(next.Preexisting, want) {
		t.Fatalf("preexisting = %v, want %v", next.Preexisting, want)
	}
}

func TestRemoveAll(t *testing.T) {
	o := Ownership{
		Preexisting: map[string][]string{"r1": {"pci"}, "r3": {"pci", "sox"}},
		Drifted:     []string{"r4"},
	}
	got := o.RemoveAll([]string{"r1", "r2", "r3"}, []string{"pci", "sox"})
	want := []ChangeGroup{
		{IDs: []string{"r2", "r4"}, Change: Change{Remove: []string{"pci", "sox"}}},
		{IDs: []string{"r1"}, Change: Change{Remove: []string{"sox"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

type memPrivate map[string][]byte

func (m memPrivate) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return m[key], nil
}

func (m memPrivate) SetKey(_ context.Context, key string, value []byte) diag.Diagnostics {
	m[key] = value
	return nil
}

func TestOwnershipRoundTrip(t *testing.T) {
	ctx := context.Background()
	p := memPrivate{}

	empty, diags := LoadOwnership(ctx, p)
	if diags.HasError() || len(empty.Preexisting) != 0 || len(empty.Drifted) != 0 {
		t.Fatalf("expected empty ownership, got %+v %v", empty, diags)
	}

	in := Ownership{
		Preexisting: map[string][]string{"r1": {"sox", "pci", "pci"}, "r2": {}},
		Drifted:     []string{"r3", "r1"},
	}
	if diags := in.Save(ctx, p); diags.HasError() {
		t.Fatal(diags)
	}

	out, diags := LoadOwnership(ctx, p)
	if diags.HasError() {
		t.Fatal(diags)
	}
	want := Ownership{
		Preexisting: map[string][]string{"r1": {"pci", "sox"}},
		Drifted:     []string{"r1", "r3"},
	}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("got %+v, want %+v", out, want)
	}
}
