package token

import (
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// ValidateDelegatedAccessToken validates an RFC 8693 delegated access token
// presented at Authorizer's OWN API.
//
// # Why this exists as a separate path
//
// Delegated tokens are stateless by design: CreateDelegatedAccessToken does not
// register them in the memory store, because a resource server verifies them
// locally against the published JWKS with no round trip here. ValidateAccessToken
// is stateful — it requires a `nonce` claim and a matching session entry, and
// compares the presented token byte-for-byte against the stored copy. A
// delegated token has no nonce, so it fails that check immediately.
//
// The consequence was that an agent could never ask Authorizer a question about
// its own delegated authority — check_permissions was unreachable with the very
// token that proves the delegation.
//
// This is deliberately NOT a branch inside ValidateAccessToken. Keeping it a
// named, separate function means the first-party path is untouched and this
// weaker path can be reviewed, tested and reasoned about on its own. Widening it
// requires editing this function, not slipping a condition into a shared one.
//
// # What is still enforced
//
//   - Signature and expiry, via ParseJWTToken.
//   - An `act` claim MUST be present. Without it the token is not delegated and
//     has no business on this path.
//   - `aud` MUST equal this server's client_id. A token minted with an RFC 8707
//     resource indicator carries that resource as its `aud` and is usable ONLY
//     there — accepting it here would be audience confusion and would make the
//     resource binding decorative. An agent that wants to call Authorizer must
//     explicitly request Authorizer as the resource.
//   - Issuer/claims via ValidateJWTClaims, and token_type must be an access token.
//   - The subject user must not be revoked. userIsRevoked is a database lookup,
//     not a session lookup, so revoking a user still stops their agents.
//
// # What is knowingly given up
//
// Per-session revocation. A first-party token dies when its session entry is
// deleted (logout, password reset); a delegated token cannot, because it was
// never stored. That is bounded by DelegatedAccessTokenTTL, which is short by
// construction, and is the same trade already accepted for resource servers.
func (p *provider) ValidateDelegatedAccessToken(gc *gin.Context, accessToken string) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	if accessToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	res, err := p.ParseJWTToken(accessToken)
	if err != nil {
		return res, err
	}

	// Must actually be a delegated token.
	if ImmediateActor(res) == "" {
		return res, fmt.Errorf(`unauthorized: not a delegated token`)
	}

	userID, ok := res["sub"].(string)
	if !ok || userID == "" {
		return res, fmt.Errorf(`unauthorized: missing sub claim`)
	}

	// Audience isolation. Anything that is not exactly this server's client_id
	// is refused — in particular a resource-indicator audience, which is an
	// absolute URI and belongs to a downstream resource server.
	aud, _ := res["aud"].(string)
	if aud != p.config.ClientID {
		if u, uErr := url.Parse(aud); uErr == nil && u.IsAbs() {
			p.dependencies.Log.Debug().Str("aud", aud).
				Msg("delegated token rejected: resource-bound audience is not valid at authorizer's own endpoints")
		}
		return res, fmt.Errorf(`unauthorized: token audience is not this server`)
	}

	if p.userIsRevoked(gc, userID) {
		p.dependencies.Log.Debug().Str("user_id", userID).Msg("delegated token rejected: user revoked")
		return res, fmt.Errorf(`unauthorized: user revoked`)
	}

	hostname := parsers.GetHost(gc)
	// ValidateJWTTokenWithoutNonce, not ValidateJWTClaims: the latter compares
	// the nonce claim unconditionally, and a delegated token carries none by
	// design (it is stateless, so there is no session nonce to bind to). Every
	// other claim — audience, issuer, subject — is still checked identically.
	if ok, vErr := p.ValidateJWTTokenWithoutNonce(res, &AuthTokenConfig{
		HostName: hostname,
		User:     &schemas.User{ID: userID},
		ClientID: aud,
	}); !ok || vErr != nil {
		return res, vErr
	}

	if res["token_type"] != constants.TokenTypeAccessToken {
		return res, fmt.Errorf(`unauthorized: invalid token type`)
	}

	return res, nil
}
