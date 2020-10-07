package common

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
	diffCounter := 0

	if len(sl1) <= len(sl2) {
		for _, val := range sl1 {
			if !StringInSlice(val, sl2) {
				diffCounter++
			}
		}
		diffCounter += len(sl2) - len(sl1)
	} else {
		for _, val := range sl2 {
			if !StringInSlice(val, sl1) {
				diffCounter++
			}
		}
		diffCounter += len(sl1) - len(sl2)
	}

	return diffCounter
}
