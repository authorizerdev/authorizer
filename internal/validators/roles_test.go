package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidRoles(t *testing.T) {
	t.Run("should return true for valid subset", func(t *testing.T) {
		assert.True(t, IsValidRoles([]string{"user"}, []string{"user", "admin"}))
	})

	t.Run("should return true for exact match", func(t *testing.T) {
		assert.True(t, IsValidRoles([]string{"user", "admin"}, []string{"user", "admin"}))
	})

	t.Run("should return false for role not in allowed list", func(t *testing.T) {
		assert.False(t, IsValidRoles([]string{"admin"}, []string{"user"}))
	})

	t.Run("should return false for escalation attempt", func(t *testing.T) {
		assert.False(t, IsValidRoles([]string{"superadmin"}, []string{"user", "admin"}))
	})

	t.Run("should return true for empty requested roles", func(t *testing.T) {
		assert.True(t, IsValidRoles([]string{}, []string{"user", "admin"}))
	})
}
