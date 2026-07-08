package token

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// DelegatedAccessTokenTTL is the fixed short lifetime of an exchanged delegation
// token (AGENTIC_DELEGATION_DESIGN DC5: 5-minute baseline). Delegation tokens
// are not refreshable — the agent re-exchanges — and sensitive-scope revocation
// is enforced out of band via /oauth/introspect, so a short TTL bounds the blast
// radius of a leaked token.
const DelegatedAccessTokenTTL = 5 * time.Minute

// accessTokenJWTType is the RFC 9068 media type stamped in the JWT header `typ`
// of an OAuth2 access token.
const accessTokenJWTType = "at+jwt"

// DelegationTokenConfig configures an RFC 8693 delegated access token.
type DelegationTokenConfig struct {
	// Subject is the `sub` claim — the user (authority source) on whose behalf
	// the actor acts. It is taken from the exchanged subject_token, never from a
	// caller-supplied parameter.
	Subject string
	// Actor is the RFC 8693 §4.1 `act` claim: {"sub": <immediate actor>, "act":
	// <prior chain>}. The caller builds this from the authenticated agent's
	// client_id plus any prior act chain carried on the subject_token.
	Actor map[string]interface{}
	// Audience is the single RFC 8707 `resource` the token is bound to; it
	// becomes the `aud` claim so the token is not replayable at another server.
	Audience string
	// Scope is the attenuated (intersected) scope set.
	Scope []string
	// ClientID is the authenticated calling agent's client_id (RFC 9068
	// `client_id` claim) — the immediate actor's registered identity.
	ClientID string
	// HostName is the issuer (`iss`).
	HostName string
}

// CreateDelegatedAccessToken mints the RFC 8693 delegation access token. Unlike
// the first-party access tokens it is stateless (not registered in the memory
// store) and its `aud` is the bound resource, not this server's client_id — it
// is validated by the downstream resource server (local JWT verification and/or
// /oauth/introspect), never by Authorizer's own ValidateAccessToken path. The
// CUSTOM_ACCESS_TOKEN_SCRIPT is intentionally not run: `act`/`client_id` are
// reserved and there is no resource-owner user object to feed the script.
func (p *provider) CreateDelegatedAccessToken(cfg *DelegationTokenConfig) (*JWTToken, error) {
	expiresAt := time.Now().Add(DelegatedAccessTokenTTL).Unix()
	claims := jwt.MapClaims{
		"iss":        cfg.HostName,
		"aud":        cfg.Audience,
		"sub":        cfg.Subject,
		"exp":        expiresAt,
		"iat":        time.Now().Unix(),
		"jti":        uuid.New().String(),
		"token_type": constants.TokenTypeAccessToken,
		"scope":      cfg.Scope,
		"client_id":  cfg.ClientID,
		"act":        cfg.Actor,
	}
	signed, err := p.signJWTToken(claims, accessTokenJWTType)
	if err != nil {
		return nil, err
	}
	return &JWTToken{Token: signed, ExpiresAt: expiresAt}, nil
}
