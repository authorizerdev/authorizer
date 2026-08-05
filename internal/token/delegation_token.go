package token

import (
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// DelegatedAccessTokenTTL is the fixed short lifetime of an exchanged delegation
// token (AGENTIC_DELEGATION_DESIGN DC5: 5-minute baseline). Delegation tokens
// are not refreshable — the agent re-exchanges.
//
// REVOCATION: none AT THE DOWNSTREAM RESOURCE SERVER. This TTL is the only
// bound there. (At Authorizer's own API the token additionally dies with the
// session it was derived from — see DelegationSessionID — but a resource server
// verifying offline against the JWKS cannot see that.) Earlier revisions of this
// comment claimed "sensitive-scope revocation is enforced out of band via
// /oauth/introspect"; that was never true and is corrected here rather than left
// as a false assurance:
//
//   - These tokens are stateless — nothing is written to the session store at
//     issuance, so there is no entry to delete.
//   - /oauth/introspect cannot report on one anyway. It answers only for a token
//     whose `aud` is the AUTHENTICATED CALLER's client_id, while a delegated
//     token's `aud` is the RFC 8707 resource URI. A resource server presenting
//     its own credentials therefore always gets {"active": false} — correct
//     per RFC 7662 §2.2 (no oracle), but useless as a revocation signal.
//
// So a delegated token is valid until it expires, full stop. Downstream
// resource servers verify it locally against /.well-known/jwks.json. Keep this
// TTL short; lengthening it directly lengthens the window in which a stolen
// token cannot be stopped.
//
// Making revocation real needs a resource registry (which client may introspect
// which resource's tokens) plus either stateful issuance or a deny-list — a
// design change, not a patch. Until that exists, do not document or rely on
// revocation for delegated tokens.
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
	// SessionID identifies the subject's session that this delegation was
	// derived from, built with DelegationSessionID. It becomes the `sid` claim
	// and is what makes the delegation revocable at Authorizer's own API — see
	// DelegationSessionID.
	//
	// Empty is permitted at mint (a subject token with no session cannot
	// produce one) but a token without it can never authenticate HERE; it
	// remains usable at the downstream resource server it was bound to.
	SessionID string
}

// DelegationSessionID encodes the memory-store coordinates of the session a
// delegation was derived from, as an opaque OIDC `sid` value (OIDC Session
// Management §3, where `sid` is defined as an opaque session identifier).
//
// Delegated tokens are stateless: nothing is written at issuance, so there is no
// entry to delete and DelegatedAccessTokenTTL is the only bound on a leaked one.
// That trade is unavoidable for a downstream resource server, which verifies the
// token offline against the JWKS and cannot consult us.
//
// It is NOT unavoidable at Authorizer's own API, and leaving it there was a real
// hole: a delegated token kept authenticating after the user logged out, reset
// their password, or had every session wiped by an admin, because none of those
// levers touch anything the token depends on. Carrying the ORIGINATING session's
// coordinates makes the delegation exactly as revocable as the credential it was
// derived from — logout of that session, or any DeleteAllUserSessions, drops the
// entry and the delegated token stops working on the next call.
//
// The format mirrors ValidateAccessToken's session-key derivation
// ("<login_method>:<user_id>|<nonce>", the login_method half omitted when the
// token carries none) so both paths address the same entry. It is deliberately
// NOT stamped as separate `nonce` and `login_method` claims: a `login_method`
// claim on a delegated token would make service/fga.go classify the caller as a
// service_account subject, silently breaking the invariant that a delegated
// token always resolves to "user:<sub>".
//
// NOTE the OIDC Back-Channel Logout token (backchannel_logout.go) also carries
// a `sid`, and sends the BARE NONCE. The two are deliberately not identical —
// that one is an outward-facing OIDC contract with relying parties and must not
// change shape — but this value ends with it, so a consumer that ever needs to
// correlate the two can match on the suffix. Both are opaque to their
// recipients; only this server interprets either.
func DelegationSessionID(loginMethod, userID, nonce string) string {
	// Trim before building, not just before testing: the result is a lookup key,
	// and a stray space baked into it would address an entry that never exists.
	loginMethod, userID, nonce = strings.TrimSpace(loginMethod), strings.TrimSpace(userID), strings.TrimSpace(nonce)
	if userID == "" || nonce == "" {
		return ""
	}
	sessionKey := userID
	if loginMethod != "" {
		sessionKey = loginMethod + ":" + userID
	}
	return sessionKey + "|" + nonce
}

// ParseDelegationSessionID splits a `sid` back into the memory-store session key
// and nonce. Split on the LAST separator: a login method never contains one and
// a nonce is a UUID, but the user id half must not be able to shift the
// boundary. Reports false for anything malformed, which callers treat as "no
// session to verify" and therefore as a denial.
func ParseDelegationSessionID(sid string) (sessionKey, nonce string, ok bool) {
	i := strings.LastIndex(sid, "|")
	if i <= 0 || i == len(sid)-1 {
		return "", "", false
	}
	return sid[:i], sid[i+1:], true
}

// CreateDelegatedAccessToken mints the RFC 8693 delegation access token. Unlike
// the first-party access tokens it is stateless (not registered in the memory
// store) and its `aud` is the bound resource, not this server's client_id — it
// is validated by the downstream resource server through local JWT verification
// against the published JWKS, never by Authorizer's own ValidateAccessToken
// path and NOT via /oauth/introspect (see DelegatedAccessTokenTTL). The
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
	// Opaque to a downstream resource server, which ignores it; the revocation
	// hook for this server. Omitted entirely when the subject had no session, so
	// the claim's presence always means "this is checkable".
	if cfg.SessionID != "" {
		claims["sid"] = cfg.SessionID
	}
	signed, err := p.signJWTToken(claims, accessTokenJWTType)
	if err != nil {
		return nil, err
	}
	return &JWTToken{Token: signed, ExpiresAt: expiresAt}, nil
}
