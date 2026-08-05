package token

import (
	"fmt"
	"net/url"
	"strings"

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
//   - `aud` MUST equal this server's own URL. A token minted with an RFC 8707
//     resource indicator carries that resource as its `aud` and is usable ONLY
//     there — accepting it here would be audience confusion and would make the
//     resource binding decorative. An agent that wants to call Authorizer must
//     explicitly request Authorizer's URL as the resource.
//   - Issuer/claims via ValidateJWTClaims, and token_type must be an access token.
//   - The subject must not be revoked or deactivated — a database lookup, not a
//     session lookup, so revoking a user still stops their agents.
//   - The session the delegation was derived from must still exist. See
//     DelegationSessionID: this is what makes logout and password reset stop a
//     delegated token here, and it is checked LAST because it is the only step
//     that touches the memory store.
//
// # What is knowingly given up
//
// The byte-for-byte comparison against a stored copy of the token. A first-party
// token is compared against the exact bytes held in its session entry; a
// delegated token is never stored, so this path can only confirm that the
// originating session is still live, not that this specific token is the one
// that session issued. The signature, the short TTL and the audience binding
// carry the rest.
func (p *provider) ValidateDelegatedAccessToken(gc *gin.Context, accessToken string) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	if accessToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}
	// Fail closed rather than panic. The audience and issuer checks below both
	// derive this server's host from the request, and parsers.GetHost does not
	// guard a nil one — an auth path must reject, never crash.
	if gc == nil || gc.Request == nil {
		return res, fmt.Errorf(`unauthorized: no request context`)
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

	hostname := parsers.GetHost(gc)

	// Audience isolation, RFC 8707. /oauth/token requires `resource` to be an
	// ABSOLUTE URI and stamps it verbatim as `aud`, so the only way to name
	// this server is to request this server's own URL as the resource. The
	// audience must therefore equal that URL — not the opaque --client-id,
	// which no resource indicator can ever be.
	//
	// Getting this wrong in the strict direction is not a safe failure: it
	// makes the delegated path unreachable by any token this deployment can
	// mint, so the feature silently does nothing. Getting it wrong in the loose
	// direction accepts a token minted for a downstream resource server, which
	// is audience confusion. Both are tested end to end through /oauth/token.
	//
	// Compared after trimming a trailing slash so "https://auth.example.com"
	// and "https://auth.example.com/" are the same audience — otherwise the
	// caller's exact spelling of the resource decides whether auth works.
	aud, _ := res["aud"].(string)
	if !sameAudience(aud, hostname) {
		if u, uErr := url.Parse(aud); uErr == nil && u.IsAbs() {
			p.dependencies.Log.Debug().Str("aud", aud).Str("expected", hostname).
				Msg("delegated token rejected: audience names a different resource server")
		}
		return res, fmt.Errorf(`unauthorized: token audience is not this server`)
	}

	if !p.delegationSubjectIsLive(gc, userID) {
		return res, fmt.Errorf(`unauthorized: delegation subject is not active`)
	}

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

	if !p.delegationSessionIsLive(res) {
		return res, fmt.Errorf(`unauthorized: originating session is no longer valid`)
	}

	return res, nil
}

// delegationSessionIsLive reports whether the session this delegation was
// derived from still exists in the memory store.
//
// This is the revocation lever. Without it a delegated token kept working after
// the delegating user logged out, reset their password, changed their email, or
// had every session wiped by an admin — none of which touch anything a stateless
// token depends on, so the only bound was DelegatedAccessTokenTTL. The `sid`
// claim (see DelegationSessionID) addresses the same memory-store entry that
// ValidateAccessToken checks for the first-party token the delegation came from,
// so every existing revocation path takes the delegation down with it, with no
// new storage, no schema change and no new revocation surface to maintain.
//
// Only the ENTRY'S EXISTENCE is checked, never its value: the entry holds the
// original subject token, not this one.
//
// Fails CLOSED on a missing or malformed `sid`. A delegated token minted from a
// subject that had no session cannot be checked, and something uncheckable must
// not authenticate at Authorizer's own API — it stays usable at the downstream
// resource server it was actually bound to, which is where it belongs.
func (p *provider) delegationSessionIsLive(claims map[string]interface{}) bool {
	if p.dependencies.MemoryStoreProvider == nil {
		return false
	}
	sid, _ := claims["sid"].(string)
	sessionKey, nonce, ok := ParseDelegationSessionID(sid)
	if !ok {
		p.dependencies.Log.Debug().
			Msg("delegated token rejected: no usable sid claim, so the originating session cannot be verified")
		return false
	}
	if _, err := p.dependencies.MemoryStoreProvider.GetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce); err != nil {
		p.dependencies.Log.Debug().Err(err).Str("session_key", sessionKey).
			Msg("delegated token rejected: originating session is gone (logout, password reset or admin revoke)")
		return false
	}
	return true
}

// sameAudience compares an audience claim with this server's URL, tolerating a
// trailing slash on either side. An empty audience never matches, so a token
// with no aud cannot pass by accident.
func sameAudience(aud, hostname string) bool {
	a := strings.TrimSuffix(strings.TrimSpace(aud), "/")
	h := strings.TrimSuffix(strings.TrimSpace(hostname), "/")
	return a != "" && h != "" && a == h
}

// delegationSubjectIsLive reports whether the subject a delegated token was
// minted for is still active.
//
// The subject is NOT always a user. RFC 8693 token exchange also accepts a
// service account as the subject, which is how a multi-hop chain is expressed
// (agent A delegates to agent B). userIsRevoked only ever looked the subject up
// as a USER, so for a service-account subject it found nothing, reported "not
// revoked", and the delegation kept working for the token's full lifetime after
// the service account had been deactivated — deactivation did not stop the
// chain it seeded.
//
// Resolution order mirrors how the token endpoint validates the subject at
// mint time (see handleTokenExchangeGrant): try user, then client.
//
// Fails CLOSED when the subject resolves to neither. A subject we cannot
// confirm is live must not authenticate — the same rule the exchange applies
// before it will seed a delegation at all.
func (p *provider) delegationSubjectIsLive(gc *gin.Context, subject string) bool {
	if p.dependencies.StorageProvider == nil || subject == "" {
		return false
	}

	if user, err := p.dependencies.StorageProvider.GetUserByID(gc, subject); err == nil && user != nil {
		if user.RevokedTimestamp != nil {
			p.dependencies.Log.Debug().Str("subject", subject).
				Msg("delegated token rejected: subject user is revoked")
			return false
		}
		return true
	}

	if client, err := p.dependencies.StorageProvider.GetClientByID(gc, subject); err == nil && client != nil {
		if !client.IsActive {
			p.dependencies.Log.Debug().Str("subject", subject).
				Msg("delegated token rejected: subject service account is deactivated")
			return false
		}
		return true
	}

	p.dependencies.Log.Debug().Str("subject", subject).
		Msg("delegated token rejected: subject resolves to neither an active user nor an active client")
	return false
}
