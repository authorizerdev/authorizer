package authorization

import (
	"unicode"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// isValidIdentifier checks that a string is safe for use in cache keys
// and database queries. Allows alphanumeric, hyphens, underscores.
// Max constants.MaxAuthzIdentifierLength characters. Empty strings are invalid.
func isValidIdentifier(s string) bool {
	if len(s) == 0 || len(s) > constants.MaxAuthzIdentifierLength {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return false
		}
	}
	return true
}
