package utils

import "sort"

// StringSlicesEqual checks if two string slices contain the same elements.
// Order does not matter - slices are sorted before comparison.
func StringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	// Make copies to avoid modifying the original slices
	leftCopy := make([]string, len(left))
	rightCopy := make([]string, len(right))

	copy(leftCopy, left)
	copy(rightCopy, right)

	sort.Strings(leftCopy)
	sort.Strings(rightCopy)

	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}

	return true
}
