package validators

// IsStringArrayEqual validates if string array are equal.
// This does check if the order is same
func IsStringArrayEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
