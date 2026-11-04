package bulk

// Unique returns items with duplicates removed, preserving first-seen order.
func Unique(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		if _, ok := seen[it]; ok {
			continue
		}
		seen[it] = struct{}{}
		out = append(out, it)
	}
	return out
}

// Difference returns the items of a that are not in b, preserving order of a.
func Difference(a, b []string) []string {
	exclude := toSet(b)
	out := make([]string, 0, len(a))
	for _, it := range a {
		if _, ok := exclude[it]; !ok {
			out = append(out, it)
		}
	}
	return out
}

// Intersection returns the items of a that are also in b, preserving order of a.
func Intersection(a, b []string) []string {
	include := toSet(b)
	out := make([]string, 0, len(a))
	for _, it := range a {
		if _, ok := include[it]; ok {
			out = append(out, it)
		}
	}
	return out
}

// ContainsAll reports whether every item of want is present in have.
func ContainsAll(have, want []string) bool {
	set := toSet(have)
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return true
}

// Chunk splits items into consecutive slices of at most size elements.
func Chunk[T any](items []T, size int) [][]T {
	if size < 1 {
		size = 1
	}
	chunks := make([][]T, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := min(start+size, len(items))
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

// SetDiff describes how a set of IDs changed between prior state and plan.
type SetDiff struct {
	Added   []string // in new, not in old
	Removed []string // in old, not in new
	Kept    []string // in both
}

// Diff compares an old and a new set of IDs.
func Diff(oldItems, newItems []string) SetDiff {
	oldItems, newItems = Unique(oldItems), Unique(newItems)
	return SetDiff{
		Added:   Difference(newItems, oldItems),
		Removed: Difference(oldItems, newItems),
		Kept:    Intersection(newItems, oldItems),
	}
}

func toSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, it := range items {
		set[it] = struct{}{}
	}
	return set
}
