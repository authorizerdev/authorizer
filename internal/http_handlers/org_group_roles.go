package http_handlers

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// orgGroupDerivedRoles resolves the role names a user holds in orgID through the
// FGA graph — roles granted transitively via group membership
// (role:<orgID>/<name>#assignee@group:<orgID>/<gid>#member) or granted to the
// user directly (role:<orgID>/<name>#assignee@user:<uid>). It is the JWT-claim
// twin of assertedGroupsForOrg (saml_idp.go): the SCIM/SSO group→role projection
// deferred by issue #692. Callers union the result onto the roles they already
// mint, so it only ever ADDS roles for the org being authenticated into.
//
// CROSS-TENANT CONTAINMENT (security-critical): a user may legitimately hold
// roles in several orgs. A token minted for orgID must NEVER carry a role
// belonging to another org — an org-B "admin" role must not leak into an org-A
// token. Containment is enforced by the org-namespace of the role object,
// mirroring assertedGroupsForOrg's two gates:
//
//	Gate 1 (namespace): the FGA object id must start with "role:<orgID>/".
//	                    Role objects are always org-namespaced and orgID is a
//	                    slash-free UUID, so a foreign org's role cannot match.
//	Gate 2 (shape, defense in depth): the stripped remainder must be a bare role
//	                    name — non-empty and containing no "/". Roles have no DB
//	                    row to verify against (unlike groups' ScimGroup row), so
//	                    the structural check is the row-of-record analog: it
//	                    rejects any object id that slipped past Gate 1 malformed.
//
// Fail-closed: no engine, or any lookup error, yields NO derived roles (never a
// partial or unscoped set). Because callers union onto their existing roles, a
// failure only ever falls back to today's behaviour — never fewer roles, never a
// blocked login.
func (h *httpProvider) orgGroupDerivedRoles(ctx context.Context, orgID string, user *schemas.User, log *zerolog.Logger) []string {
	if h.AuthzEngine == nil {
		return nil
	}
	l := log.With().Str("func", "orgGroupDerivedRoles").Logger()
	start := time.Now()
	objects, err := h.AuthzEngine.ListObjects(ctx, "user:"+user.ID, "assignee", "role")
	metrics.ObserveFgaCheckDuration(metrics.FgaOpDerivedRoles, time.Since(start).Seconds())
	if err != nil {
		// No model, no role type/relation, engine down — derive nothing.
		metrics.RecordFgaCheck(metrics.FgaOpDerivedRoles, metrics.FgaResultError)
		l.Debug().Err(err).Msg("role lookup failed, deriving no roles")
		return nil
	}
	metrics.RecordFgaCheck(metrics.FgaOpDerivedRoles, metrics.FgaResultSuccess)
	prefix := "role:" + orgID + "/"
	seen := map[string]bool{}
	var names []string
	for _, obj := range objects {
		name, ok := strings.CutPrefix(obj, prefix) // Gate 1: namespace.
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" || strings.Contains(name, "/") { // Gate 2: bare name.
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

// unionRoles returns base with every role in extra not already present appended,
// preserving base's order. The result is always a superset of base — the
// additive, never-fewer-roles guarantee the org role projection relies on.
func unionRoles(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]bool, len(base))
	for _, r := range base {
		seen[r] = true
	}
	out := base
	for _, r := range extra {
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}
