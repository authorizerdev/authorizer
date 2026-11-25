package config

import "strings"

// IsConfiguredRole reports whether role is currently configured on this
// instance, i.e. listed in --roles, --protected-roles or --default-roles.
func (c *Config) IsConfiguredRole(role string) bool {
	if c == nil || role == "" {
		return false
	}
	for _, set := range [][]string{c.Roles, c.ProtectedRoles, c.DefaultRoles} {
		for _, r := range set {
			if r == role {
				return true
			}
		}
	}
	return false
}

// EffectiveUserRoles returns the roles stored on a user record (the
// comma-separated schemas.User.Roles value) that are still configured.
//
// A role removed from --roles stays in stored user records; it is filtered
// here at read time instead of being rewritten in storage, so re-adding the
// role to the configuration restores it for every user that held it (#425).
func (c *Config) EffectiveUserRoles(storedRoles string) []string {
	out := []string{}
	for _, r := range strings.Split(storedRoles, ",") {
		r = strings.TrimSpace(r)
		if c.IsConfiguredRole(r) {
			out = append(out, r)
		}
	}
	return out
}

// WithoutStaleUserRoles removes from roles every role that the user record
// holds but that is no longer configured.
//
// It deliberately does NOT intersect roles with the configured set: a token's
// roles can also carry org-group-derived roles resolved through the FGA engine
// (see http_handlers/org_group_roles.go), which are never listed in --roles and
// never written to the user record. Only roles sourced from the user record are
// governed by --roles, so only those are dropped.
func (c *Config) WithoutStaleUserRoles(roles []string, storedRoles string) []string {
	stale := map[string]bool{}
	for _, r := range strings.Split(storedRoles, ",") {
		r = strings.TrimSpace(r)
		if r != "" && !c.IsConfiguredRole(r) {
			stale[r] = true
		}
	}
	if len(stale) == 0 {
		return roles
	}
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		if !stale[r] {
			out = append(out, r)
		}
	}
	return out
}
