package token

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLeftMostHalfHash pins OIDC Core §3.1.3.6 (at_hash) / §3.3.2.11 (c_hash):
// the digest must be the one implied by the ID token's `alg`, not always
// SHA-256. An RP that follows the spec recomputes the hash from `alg`, so a
// mismatched digest silently disables the token-substitution check the claim
// exists to provide.
func TestLeftMostHalfHash(t *testing.T) {
	t.Parallel()

	const token = "an-access-token-value"

	sha256Half := func(v string) string {
		sum := sha256.Sum256([]byte(v))
		return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2])
	}
	sha384Half := func(v string) string {
		sum := sha512.Sum384([]byte(v))
		return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2])
	}
	sha512Half := func(v string) string {
		sum := sha512.Sum512([]byte(v))
		return base64.RawURLEncoding.EncodeToString(sum[:len(sum)/2])
	}

	for _, tc := range []struct {
		alg  string
		want string
	}{
		{"HS256", sha256Half(token)},
		{"RS256", sha256Half(token)},
		{"ES256", sha256Half(token)},
		{"HS384", sha384Half(token)},
		{"RS384", sha384Half(token)},
		{"ES384", sha384Half(token)},
		{"HS512", sha512Half(token)},
		{"RS512", sha512Half(token)},
		{"ES512", sha512Half(token)},
	} {
		t.Run(tc.alg, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, leftMostHalfHash(token, tc.alg))
		})
	}

	t.Run("the 384 and 512 families are actually distinct from SHA-256", func(t *testing.T) {
		t.Parallel()
		// Guards the regression directly: before the fix all three were equal,
		// because every alg went through SHA-256.
		assert.NotEqual(t, leftMostHalfHash(token, "RS256"), leftMostHalfHash(token, "RS384"))
		assert.NotEqual(t, leftMostHalfHash(token, "RS256"), leftMostHalfHash(token, "RS512"))
		assert.NotEqual(t, leftMostHalfHash(token, "RS384"), leftMostHalfHash(token, "RS512"))
	})

	t.Run("left-most half, not the whole digest", func(t *testing.T) {
		t.Parallel()
		// RS256 -> SHA-256 -> 32-byte digest -> 16 bytes kept -> 22 base64url chars.
		raw, err := base64.RawURLEncoding.DecodeString(leftMostHalfHash(token, "RS256"))
		assert.NoError(t, err)
		assert.Len(t, raw, 16, "RS256 keeps the left-most 128 bits")

		raw, err = base64.RawURLEncoding.DecodeString(leftMostHalfHash(token, "RS512"))
		assert.NoError(t, err)
		assert.Len(t, raw, 32, "RS512 keeps the left-most 256 bits")
	})
}
