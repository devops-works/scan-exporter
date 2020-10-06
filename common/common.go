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
