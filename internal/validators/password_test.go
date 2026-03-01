package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidPassword(t *testing.T) {
	t.Run("strong password enabled", func(t *testing.T) {
		// isStrongPasswordDisabled = false means strong password IS enforced
		assert.NoError(t, IsValidPassword("Password@123", false))
		assert.Error(t, IsValidPassword("simple", false), "should reject password without uppercase/digit/special")
		assert.Error(t, IsValidPassword("SIMPLE", false), "should reject password without lowercase/digit/special")
		assert.Error(t, IsValidPassword("Simple", false), "should reject password without digit/special")
		assert.Error(t, IsValidPassword("Simple1", false), "should reject password without special char")
		assert.Error(t, IsValidPassword("ab", false), "should reject short password")
		assert.Error(t, IsValidPassword("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false), "should reject password over 36 chars")
	})

	t.Run("strong password disabled", func(t *testing.T) {
		// isStrongPasswordDisabled = true means strong password is NOT enforced
		assert.NoError(t, IsValidPassword("simple", true), "should accept lowercase-only password when strong password disabled")
		assert.NoError(t, IsValidPassword("123456", true), "should accept digits-only password when strong password disabled")
		assert.NoError(t, IsValidPassword("Password@123", true), "should accept strong password even when disabled")
		assert.Error(t, IsValidPassword("ab", true), "should still reject short password")
		assert.Error(t, IsValidPassword("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true), "should still reject password over 36 chars")
	})
}
