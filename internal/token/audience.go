package token

import (
	"github.com/golang-jwt/jwt/v4"
)

// AudienceMatches reports whether the JWT `aud` claim contains the
// expected audience. RFC 7519 §4.1.3 allows `aud` to be either a single
// case-sensitive string or an array of case-sensitive strings, so naive
// `==` comparisons miss the array shape entirely. JSON-decoded JWT
// claims may surface as any of the following Go types depending on the
// upstream parser, so this helper accepts all of them:
//
//   - string
//   - []string
//   - []interface{} (default json.Unmarshal target for arrays)
//   - jwt.ClaimStrings (typed wrapper from golang-jwt/jwt/v4)
//
// Returns false for nil, empty arrays, or any other type. Comparison
// is exact — no scheme normalization or case folding — to match the
// RFC 7519 case-sensitive requirement.
func AudienceMatches(aud interface{}, expected string) bool {
	if aud == nil || expected == "" {
		return false
	}
	switch v := aud.(type) {
	case string:
		return v == expected
	case []string:
		for _, a := range v {
			if a == expected {
				return true
			}
		}
	case jwt.ClaimStrings:
		for _, a := range v {
			if a == expected {
				return true
			}
		}
	case []interface{}:
		for _, a := range v {
			if s, ok := a.(string); ok && s == expected {
				return true
			}
		}
	}
	return false
}
