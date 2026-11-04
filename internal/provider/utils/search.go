package utils

// Local name search for the list data sources. The list endpoints do not accept
// search together with the afterId cursor or the other filters, so the data
// sources read every page and search the names here.

import (
	"cmp"
	"slices"
	"strings"
)

// Search ranks, best first.
const (
	SearchRankExact = iota
	SearchRankCaseInsensitive
	SearchRankPartial
	SearchRankNone
)

// SearchRank ranks name against term: exact match, case-insensitive match,
// case-insensitive partial match, or no match (SearchRankNone).
func SearchRank(name, term string) int {
	switch {
	case name == term:
		return SearchRankExact
	case strings.EqualFold(name, term):
		return SearchRankCaseInsensitive
	case strings.Contains(strings.ToLower(name), strings.ToLower(term)):
		return SearchRankPartial
	default:
		return SearchRankNone
	}
}

// SearchByName returns the items whose name matches term, best match first:
// exact matches, then case-insensitive matches, then partial matches. Items of
// the same rank keep their input order. An empty term returns items unchanged.
func SearchByName[T any](items []T, term string, nameOf func(T) string) []T {
	if term == "" {
		return items
	}

	type ranked struct {
		rank int
		item T
	}

	matches := make([]ranked, 0, len(items))
	for _, it := range items {
		if rank := SearchRank(nameOf(it), term); rank != SearchRankNone {
			matches = append(matches, ranked{rank, it})
		}
	}
	slices.SortStableFunc(matches, func(a, b ranked) int { return cmp.Compare(a.rank, b.rank) })

	out := make([]T, len(matches))
	for i, m := range matches {
		out[i] = m.item
	}
	return out
}
