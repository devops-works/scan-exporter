package common

import (
	"sort"
)

// StringInSlice checks if a string appears in a slice.
func StringInSlice(s string, sl []string) bool {
	for _, v := range sl {
		if v == s {
			return true
		}
	}
	return false
}

// CompareStringSlices checks if two slices are equal.
// It returns the number of different items.
func CompareStringSlices(sl1, sl2 []string) int {
	sort.Strings(sl1)
	sort.Strings(sl2)

	newports := []string{}
	missingports := []string{}

	for _, v := range sl2 {
		if !StringInSlice(v, sl1) {
			newports = append(newports, v)
		}
	}

	for _, v := range sl1 {
		if !StringInSlice(v, sl2) {
			missingports = append(missingports, v)
		}
	}

	return len(newports) + len(missingports)
}
