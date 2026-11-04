package utils

import (
	"slices"
	"testing"
)

func TestSearchRank(t *testing.T) {
	for _, tc := range []struct {
		name, term string
		want       int
	}{
		{"Admin", "Admin", SearchRankExact},
		{"admin", "Admin", SearchRankCaseInsensitive},
		{"ADMIN", "admin", SearchRankCaseInsensitive},
		{"AdministratorAccess", "admin", SearchRankPartial},
		{"db-admin-ro", "ADMIN", SearchRankPartial},
		{"Reader", "admin", SearchRankNone},
		{"", "admin", SearchRankNone},
	} {
		if got := SearchRank(tc.name, tc.term); got != tc.want {
			t.Errorf("SearchRank(%q, %q) = %d, want %d", tc.name, tc.term, got, tc.want)
		}
	}
}

func TestSearchByName(t *testing.T) {
	type item struct{ Name string }
	nameOf := func(it item) string { return it.Name }

	// Input is in API order (by ID).
	in := []item{
		{"db-admin-ro"},
		{"Reader"},
		{"admin"},
		{"Admin"},
		{"AdministratorAccess"},
		{"ADMIN"},
	}

	var got []string
	for _, it := range SearchByName(in, "Admin", nameOf) {
		got = append(got, it.Name)
	}

	// Exact, then case-insensitive, then partial; ties keep input order.
	want := []string{"Admin", "admin", "ADMIN", "db-admin-ro", "AdministratorAccess"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if out := SearchByName(in, "nothing", nameOf); len(out) != 0 {
		t.Fatalf("expected no matches, got %d", len(out))
	}
	if out := SearchByName(in, "", nameOf); len(out) != len(in) {
		t.Fatalf("empty term: got %d items, want %d", len(out), len(in))
	}
}
