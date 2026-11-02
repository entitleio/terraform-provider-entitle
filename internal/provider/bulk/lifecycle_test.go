package bulk

import (
	"context"
	"reflect"
	"testing"
)

func TestGroupBy(t *testing.T) {
	got := GroupBy(map[string]string{"b": "x", "a": "x", "c": "y", "d": ""}, func(v string) string { return v })
	want := []Group[string]{
		{IDs: []string{"a", "b"}, Value: "x"},
		{IDs: []string{"c"}, Value: "y"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPlanAdditiveInspectsOnlyWhatIsNeeded(t *testing.T) {
	var calls []map[string][]string
	inspect := func(_ context.Context, want map[string][]string) (map[string]Presence, error) {
		calls = append(calls, want)
		out := map[string]Presence{}
		for id := range want {
			out[id] = Presence{Exists: true, Values: []string{"pci"}}
		}
		return out, nil
	}

	// No additive values: nothing is read.
	if _, err := PlanAdditive(context.Background(), Additive{NewIDs: []string{"r1"}}, inspect); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no reads, got %v", calls)
	}

	// New IDs: one batched read for all of them; found values become pre-existing.
	p, err := PlanAdditive(context.Background(), Additive{NewIDs: []string{"r1", "r2"}, NewValues: []string{"pci", "sox"}}, inspect)
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string][]string{{"r1": {"pci", "sox"}, "r2": {"pci", "sox"}}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("reads = %v, want %v", calls, want)
	}
	if !reflect.DeepEqual(p.Next.Preexisting, map[string][]string{"r1": {"pci"}, "r2": {"pci"}}) {
		t.Fatalf("preexisting = %v", p.Next.Preexisting)
	}
	if !reflect.DeepEqual(p.ChangesByID["r1"], Change{Add: []string{"sox"}}) {
		t.Fatalf("change = %+v", p.ChangesByID["r1"])
	}
}

func TestRefreshMismatchIsDrift(t *testing.T) {
	calls := 0
	inspect := func(_ context.Context, want map[string][]string) (map[string]Presence, error) {
		calls++
		out := map[string]Presence{}
		for id := range want {
			if id != "gone" { // missing entities are simply left out
				out[id] = Presence{Exists: true, Mismatch: id == "changed"}
			}
		}
		return out, nil
	}
	inSync, next, err := Refresh(context.Background(), []string{"ok", "changed", "gone"}, Ownership{}, nil, inspect)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected one batched read, got %d", calls)
	}
	if !reflect.DeepEqual(inSync, []string{"ok"}) || !reflect.DeepEqual(next.Drifted, []string{"changed"}) {
		t.Fatalf("in sync = %v, drifted = %v", inSync, next.Drifted)
	}
}

func TestSortedKeys(t *testing.T) {
	if got := SortedKeys(map[string]int{"b": 1, "a": 2}); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("got %v", got)
	}
}
