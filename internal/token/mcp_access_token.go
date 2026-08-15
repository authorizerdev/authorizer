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
// matches the presented token, subject liveness, issuer/claims and token type
// all come from validateStatefulAccessToken — the same core ValidateAccessToken
// uses. MCP is NOT a weaker path: it differs from the first-party check in
// exactly one rule, the audience, and there it is the stricter of the two.
//
// # RFC 8693 delegated tokens ARE accepted, as a fallback
//
// A delegated token is stateless by design — no nonce, no session entry — so it
// always fails the stateful core above. It is then retried against
// ValidateDelegatedAccessTokenForResource, which enforces every other check plus
// an EXACT match on this same `resource`.
//
// This is what lets an agent holding "agent X acting for user Y" authority ask
// Authorizer about that authority through the MCP tools it was granted. Without
// it, check_permissions was unreachable with the very token that proves the
// delegation, and agent-delegation over MCP was a stdio-only story.
//
// Three properties make the fallback safe rather than a hole:
//
//   - Ordered as a FALLBACK, not a branch. A first-party MCP token is validated
//     exactly as before and never touches the weaker path.
//   - Gated on the `act` claim. Only a token that actually carries an actor is
//     retried, so an ordinary wrong-audience login token is rejected once
//     instead of paying a second full validation (JWT parse, session lookup,
//     subject-liveness DB read) on the hot rejection path of an
//     internet-facing endpoint. The claims are read from the stateful attempt's
//     own signature-verified parse, so this costs nothing extra; a wrong hint
//     could only cause a rejection, never an acceptance, because the delegated
//     validator re-parses and re-verifies independently.
//   - The audience match stays EXACT. A delegated token bound to the bare
//     server URL — the kind that works at /graphql today — is still refused
//     here, and an MCP-bound one is still refused there. See the bijection note
//     on ValidateDelegatedAccessTokenForResource.
//
// What the delegated path gives up relative to the stateful one is the
// byte-for-byte comparison against a stored copy of the token. The compensating
// controls are the signature, the 5-minute DelegatedAccessTokenTTL, the exact
// audience binding, and delegationSessionIsLive — so logout, password reset and
// admin revoke still take an agent's access down with the user's session.
func (p *provider) ValidateMCPAccessToken(gc *gin.Context, accessToken string, resource string) (map[string]interface{}, error) {
	if resource == "" {
		// Fail closed. An empty expected audience would make the comparison
		// below accept-nothing via sameAudience, but relying on that is too
		// subtle for an auth path: say no explicitly.
		return map[string]interface{}{}, fmt.Errorf(`unauthorized: no mcp resource configured`)
	}
	claims, err := p.validateStatefulAccessToken(gc, accessToken, func(aud string) error {
		if !sameAudience(aud, resource) {
			p.dependencies.Log.Debug().Str("aud", aud).Str("expected", resource).
				Msg("access token rejected at mcp: audience names a different resource")
			return fmt.Errorf(`unauthorized: token audience is not this mcp server`)
		}
		return nil
	})
	if err == nil {
		return claims, nil
	}
	// Not a delegated token: report the stateful failure as-is. ImmediateActor
	// reads the claims validateStatefulAccessToken already parsed and
	// signature-verified; an empty map (parse failure) yields "" and stops here.
	if ImmediateActor(claims) == "" {
		return claims, err
	}
	return p.ValidateDelegatedAccessTokenForResource(gc, accessToken, resource)
}
