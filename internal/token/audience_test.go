package token

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestAudienceMatches(t *testing.T) {
	const expected = "client-123"

	t.Run("string match", func(t *testing.T) {
		assert.True(t, AudienceMatches("client-123", expected))
	})

	t.Run("string mismatch", func(t *testing.T) {
		assert.False(t, AudienceMatches("other", expected))
	})

	t.Run("[]string contains", func(t *testing.T) {
		assert.True(t, AudienceMatches([]string{"a", "client-123", "b"}, expected))
	})

	t.Run("[]string missing", func(t *testing.T) {
		assert.False(t, AudienceMatches([]string{"a", "b"}, expected))
	})

	t.Run("[]interface{} contains", func(t *testing.T) {
		assert.True(t, AudienceMatches([]interface{}{"a", "client-123"}, expected))
	})

	t.Run("[]interface{} with non-string entries", func(t *testing.T) {
		assert.False(t, AudienceMatches([]interface{}{1, 2.5, true}, expected))
	})

	t.Run("[]interface{} mixed match", func(t *testing.T) {
		assert.True(t, AudienceMatches([]interface{}{1, "client-123"}, expected))
	})

	t.Run("jwt.ClaimStrings contains", func(t *testing.T) {
		assert.True(t, AudienceMatches(jwt.ClaimStrings{"a", "client-123"}, expected))
	})

	t.Run("jwt.ClaimStrings missing", func(t *testing.T) {
		assert.False(t, AudienceMatches(jwt.ClaimStrings{"a"}, expected))
	})

	t.Run("nil aud", func(t *testing.T) {
		assert.False(t, AudienceMatches(nil, expected))
	})

	t.Run("empty string aud", func(t *testing.T) {
		assert.False(t, AudienceMatches("", expected))
	})

	t.Run("empty []string", func(t *testing.T) {
		assert.False(t, AudienceMatches([]string{}, expected))
	})

	t.Run("empty expected", func(t *testing.T) {
		assert.False(t, AudienceMatches("client-123", ""))
	})

	t.Run("unsupported type", func(t *testing.T) {
		assert.False(t, AudienceMatches(42, expected))
	})
}
