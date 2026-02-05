package utils

import "sort"

// StringSlicesEqual checks if two string slices contain the same elements.
// Order does not matter - slices are sorted before comparison.
func StringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Make copies to avoid modifying the original slices
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}
