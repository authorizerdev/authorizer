package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEffectiveUserRoles(t *testing.T) {
	c := &Config{
		Roles:          []string{"user", "admin"},
		ProtectedRoles: []string{"super"},
		DefaultRoles:   []string{"member"},
	}
	assert.Equal(t, []string{"user", "super", "member"}, c.EffectiveUserRoles("user,removed,super,member"))
	assert.Equal(t, []string{}, c.EffectiveUserRoles(""))
	assert.Equal(t, []string{}, c.EffectiveUserRoles("removed"))
}

func TestWithoutStaleUserRoles(t *testing.T) {
	c := &Config{Roles: []string{"user"}}
	// "removed" is on the user record but no longer configured: dropped.
	// "org_admin" is not on the user record (FGA-derived): kept.
	assert.Equal(t, []string{"user", "org_admin"},
		c.WithoutStaleUserRoles([]string{"user", "removed", "org_admin"}, "user,removed"))
	// Nothing stale: input returned as is.
	assert.Equal(t, []string{"user", "org_admin"},
		c.WithoutStaleUserRoles([]string{"user", "org_admin"}, "user"))
}
