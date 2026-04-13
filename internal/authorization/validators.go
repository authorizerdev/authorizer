package authorization

import "unicode"

// isValidIdentifier checks that a string is safe for use in cache keys
// and database queries. Allows alphanumeric, hyphens, underscores.
// Max 100 characters. Empty strings are invalid.
func isValidIdentifier(s string) bool {
	if len(s) == 0 || len(s) > 100 {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return false
		}
	}
	return true
}
