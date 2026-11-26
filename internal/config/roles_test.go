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

func TestIsConfiguredRole(t *testing.T) {
	c := &Config{
		Roles:          []string{"user"},
		ProtectedRoles: []string{"super"},
		// pflag's StringSliceVar does not trim: --default-roles "user, member".
		DefaultRoles: []string{"user", " member"},
	}
	assert.True(t, c.IsConfiguredRole("user"))
	assert.True(t, c.IsConfiguredRole("super"), "protected roles are configured")
	assert.True(t, c.IsConfiguredRole("member"), "default roles are configured, whitespace ignored")
	assert.True(t, c.IsConfiguredRole(" member "))
	assert.False(t, c.IsConfiguredRole("removed"))
	assert.False(t, c.IsConfiguredRole(""))
	assert.False(t, c.IsConfiguredRole("  "))
	assert.Equal(t, []string{"user", "member"}, c.EffectiveUserRoles("user, member,removed"))
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
