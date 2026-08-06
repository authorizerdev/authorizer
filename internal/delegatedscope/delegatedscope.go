// Package delegatedscope decides what an RFC 8693 DELEGATED token — an agent
// acting on behalf of a user — is permitted to do at Authorizer's own API.
//
// # Why this exists
//
// A delegated token carries a `scope` claim that the token endpoint computes as
//
//	subject_token.scope ∩ agent.allowed_scopes ( ∩ requested )
//
// That attenuation is the entire point of the exchange, and until this package
// existed nothing consulted it. Once a delegated token could authenticate at
// Authorizer's own API, it reached EVERY first-party operation with the scope
// ignored: an agent an operator granted `openid` for a downstream MCP server
// could read the user's profile, mutate the account, and deactivate it.
//
// # The model
//
// Per-operation scope enforcement is how OAuth answers "what may this token
// do", and how every major authorization server gates its own API (Auth0's
// Management API scopes, Microsoft Graph permissions, Okta's `okta.*` scopes).
// RFC 8693 returns `scope` in the exchange response precisely so that someone
// enforces it; RFC 6750 §3.1 defines the failure as `insufficient_scope`.
//
// # Scoped to delegated callers, deliberately
//
// First-party tokens are NOT gated here, and that is not an oversight:
// `login` accepts a caller-supplied `scope` with no allow-list (see
// service.Login), so a first-party scope is a hint, not a boundary — enforcing
// it would break existing clients while granting no security.
//
// For a DELEGATED token the same claim IS a boundary, because the ceiling comes
// from `agent.allowed_scopes`, which only an admin can set. A sensitive
// operation therefore needs BOTH halves: the user's own token must carry the
// scope, and the operator must have granted the agent a ceiling that includes
// it. Neither party can widen an agent alone. That two-party requirement is the
// same shape as Microsoft's delegated permissions, where effective access is
// the app's granted permission intersected with the signed-in user's.
//
// # Fail closed
//
// An operation absent from the table is DENIED to delegated callers. New
// operations are therefore unreachable by agents until someone deliberately
// adds them, which is the opposite of an allowlist that silently widens when a
// contributor forgets it.
package delegatedscope

import "strings"

// Scope names required by operations beyond the universally-held `openid`.
//
// Deliberately few. Each one an operator can add to an agent's
// `allowed_scopes` to widen it, and each one the delegating user must also
// carry for the intersection to yield it.
const (
	// ScopeOpenID is held by every token this server mints, so operations
	// requiring only this are reachable by any delegated caller. Reserved for
	// READ-ONLY identity and permission queries — the questions an agent must
	// be able to ask to function at all.
	ScopeOpenID = "openid"
	// ScopeProfileWrite permits mutating the delegating user's profile.
	ScopeProfileWrite = "authorizer:profile:write"
	// ScopeAccountDelete permits deactivating the delegating user's account.
	// Separate from profile:write because it is irreversible by the agent.
	ScopeAccountDelete = "authorizer:account:delete"
)

// operation binds one logical API operation to the scope a delegated caller
// must hold, across every transport that exposes it.
//
// GraphQL field names and gRPC method names are listed EXPLICITLY rather than
// derived from one another. A camel-to-snake conversion looks tidy until
// `ValidateJWTToken` renders as `validate_j_w_t_token`, and a silent miss here
// fails OPEN on whichever transport the conversion got wrong. Two columns in
// one table make a mismatch visible to the reader.
type operation struct {
	graphQL string
	grpc    string
	scope   string
}

// table is the complete set of operations a delegated token may reach.
// Anything not listed is denied — see the package comment.
var table = []operation{
	// Read-only identity and permission queries. These are what an agent needs
	// to answer "may I?" and "on whose behalf?", and are exactly the tool set
	// the built-in MCP server exposes.
	{graphQL: "check_permissions", grpc: "CheckPermissions", scope: ScopeOpenID},
	{graphQL: "list_permissions", grpc: "ListPermissions", scope: ScopeOpenID},
	{graphQL: "profile", grpc: "Profile", scope: ScopeOpenID},
	{graphQL: "meta", grpc: "Meta", scope: ScopeOpenID},

	// Mutating operations. Unreachable by default: the scopes below are not in
	// any client's default request, so the intersection yields them only when
	// an operator has deliberately widened BOTH the agent's ceiling and the
	// user's own token.
	{graphQL: "update_profile", grpc: "UpdateProfile", scope: ScopeProfileWrite},
	{graphQL: "deactivate_account", grpc: "DeactivateAccount", scope: ScopeAccountDelete},
}

var (
	byGraphQL = map[string]string{}
	byGRPC    = map[string]string{}
)

func init() {
	for _, op := range table {
		byGraphQL[op.graphQL] = op.scope
		byGRPC[op.grpc] = op.scope
	}
}

// RequiredForGraphQL returns the scope a delegated caller needs for a root
// GraphQL field, and whether the field is reachable by one at all.
func RequiredForGraphQL(field string) (string, bool) {
	s, ok := byGraphQL[field]
	return s, ok
}

// RequiredForGRPC returns the scope a delegated caller needs for a gRPC method,
// and whether the method is reachable by one at all.
//
// fullMethod is the interceptor's "/package.Service/Method" form; only the
// trailing method name is significant, so the same table serves the REST
// gateway and the MCP server, both of which dispatch through gRPC.
func RequiredForGRPC(fullMethod string) (string, bool) {
	if i := strings.LastIndex(fullMethod, "/"); i >= 0 {
		fullMethod = fullMethod[i+1:]
	}
	s, ok := byGRPC[fullMethod]
	return s, ok
}

// Satisfied reports whether a token's scope claim contains the required scope.
//
// Exact string match per element, never a prefix or substring test: a caller
// holding `authorizer:profile:write:nothing` must not satisfy
// `authorizer:profile:write`, and `openid2` must not satisfy `openid`.
func Satisfied(tokenScopes []string, required string) bool {
	for _, s := range tokenScopes {
		if strings.TrimSpace(s) == required {
			return true
		}
	}
	return false
}
