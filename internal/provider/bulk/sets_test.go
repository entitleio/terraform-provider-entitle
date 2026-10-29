package bulk

import (
	"reflect"
	"testing"
)

func TestChunk(t *testing.T) {
	got := Chunk([]string{"a", "b", "c", "d", "e"}, 2)
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if got := Chunk([]string{}, 10); len(got) != 0 {
		t.Fatalf("expected no chunks, got %v", got)
	}
}

func TestDiff(t *testing.T) {
	got := Diff([]string{"a", "b", "c"}, []string{"b", "c", "d", "d"})
	want := SetDiff{Added: []string{"d"}, Removed: []string{"a"}, Kept: []string{"b", "c"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestContainsAll(t *testing.T) {
	if !ContainsAll([]string{"a", "b", "c"}, []string{"c", "a"}) {
		t.Fatal("expected true")
	}
	if ContainsAll([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("expected false")
	}
}
