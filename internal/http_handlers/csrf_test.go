package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/validators"
)

// The CORS/CSRF wildcard predicate and validators must never disagree. pflag
// does not trim the elements of a comma-separated flag, so --allowed-origins
// "https://app.acme.com, *" yields " *". When this function compared the raw
// string it called that NOT a wildcard while validators called it one, so
// CORSMiddleware took the explicit-allowlist branch and reflected the caller's
// own Origin back with Access-Control-Allow-Credentials: true. Both layers of
// CSRF defence fell with it.
func TestWildcardPredicateAgreesWithValidators(t *testing.T) {
	for _, spelling := range []string{"*", " *", "* ", "\t*", "https://*", "http://*", "*/", "*:80", "*:443"} {
		allowed := []string{"https://app.acme.com", spelling}
		assert.True(t, isWildcardOrigins(allowed),
			"%q behaves as a wildcard in validators, so it must here too — "+
				"otherwise CORS reflects any origin with credentials", spelling)
		assert.True(t, validators.IsWildcardOrigin(spelling), "%q", spelling)
	}
	// An explicit allowlist must still reach the credentialed branch.
	assert.False(t, isWildcardOrigins([]string{"https://app.acme.com", "*.acme.com"}))
}
