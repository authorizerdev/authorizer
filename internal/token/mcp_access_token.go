package token

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// ValidateMCPAccessToken validates an access token presented at Authorizer's
// own MCP surface (POST /mcp), where Authorizer acts as an OAuth 2.1 resource
// server for itself.
//
// # The audience is the whole point
//
// The MCP specification requires a client to name the MCP server as the RFC 8707
// `resource` when it asks for a token, and requires the server to accept only
// tokens issued for itself:
//
//	"MCP servers MUST only accept tokens specifically intended for themselves and
//	MUST reject tokens that do not include them in the audience claim."
//
// `resource` becomes the token's `aud` at issuance (see accessTokenAudience), so
// enforcing that here is what makes the binding real rather than decorative. A
// token minted for any other resource — including one minted for Authorizer's own
// client_id, which is what every ordinary login produces — is rejected.
//
// This is the exact mirror image of firstPartyAudienceOK, which rejects every
// resource-bound audience at /userinfo, GraphQL, gRPC and REST. Between the two
// rules, a token is accepted at exactly one surface: the one it names. Neither
// rule has an "or" in it, and that is deliberate — an MCP token must not be
// replayable at /graphql, and a login token must not be replayable at /mcp.
//
// # resource is caller-supplied, never request-derived
//
// The canonical resource URI is passed in by the caller, which computes it ONCE
// at wiring time from the operator-configured --url. It is deliberately not
// derived from the request: parsers.GetHost falls back to request headers when
// --url is unset, and an audience check against an attacker-controllable header
// authenticates anyone. Startup refuses to enable MCP without --url; this
// signature is the second lock on the same door.
//
// # Everything else is the ordinary stateful check
//
// Signature, expiry, a live memory-store session entry whose stored digest
// matches the presented token, issuer/claims and token type all come from
// validateStatefulAccessToken — the same core ValidateAccessToken uses. MCP is
// NOT a weaker path: it differs from the first-party check in the audience rule
// (stricter) and in resolving subject liveness as user-then-client (also
// stricter — see subjectIsLive; a deactivated service account is rejected here
// even though the first-party path would still accept its token).
//
// # What is deliberately NOT accepted
//
// RFC 8693 delegated tokens. They are stateless by design — no nonce, no session
// entry — so they fail the core's session lookup, and ValidateDelegatedAccessToken
// requires `aud` to equal the bare server URL rather than the /mcp resource. An
// agent therefore cannot yet reach /mcp with a delegated token. That is a scoping
// decision, not an oversight: the delegated path gives up the byte-for-byte
// comparison against a stored token, and widening it is a deliberate edit to that
// function (see its doc comment), not a side effect of adding a transport.
func (p *provider) ValidateMCPAccessToken(gc *gin.Context, accessToken string, resource string) (map[string]interface{}, error) {
	if resource == "" {
		// Fail closed. An empty expected audience would make the comparison
		// below accept-nothing via sameAudience, but relying on that is too
		// subtle for an auth path: say no explicitly.
		return map[string]interface{}{}, fmt.Errorf(`unauthorized: no mcp resource configured`)
	}
	return p.validateStatefulAccessToken(gc, accessToken,
		func(aud string) error {
			if !sameAudience(aud, resource) {
				p.dependencies.Log.Debug().Str("aud", aud).Str("expected", resource).
					Msg("access token rejected at mcp: audience names a different resource")
				return fmt.Errorf(`unauthorized: token audience is not this mcp server`)
			}
			return nil
		},
		p.subjectIsLive,
	)
}
