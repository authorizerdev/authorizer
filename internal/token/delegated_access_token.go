package token

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/storage"
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
//
//   - An `act` claim MUST be present. Without it the token is not delegated and
//     has no business on this path.
//
//   - `aud` MUST equal this server's own URL. A token minted with an RFC 8707
//     resource indicator carries that resource as its `aud` and is usable ONLY
//     there — accepting it here would be audience confusion and would make the
//     resource binding decorative. An agent that wants to call Authorizer must
//     explicitly request Authorizer's URL as the resource.
//
//     The MCP transport needs the SAME checks against a DIFFERENT audience
//     ("<url>/mcp"), and takes ValidateDelegatedAccessTokenForResource rather
//     than a second audience accepted here. Read that function before changing
//     this one: the two entry points existing separately is what keeps a token
//     valid at exactly one surface.
//
//   - Issuer/claims via ValidateJWTClaims, and token_type must be an access token.
//
//   - The subject must not be revoked or deactivated — a database lookup, not a
//     session lookup, so revoking a user still stops their agents.
//
//   - The session the delegation was derived from must still exist. See
//     DelegationSessionID: this is what makes logout and password reset stop a
//     delegated token here.
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
	// The first-party audience: this server's own URL. Unchanged behaviour —
	// every caller of this function reaches Authorizer's own API surface
	// (/graphql, /v1/*, gRPC) via GetUserIDFromSessionOrAccessToken.
	return p.validateDelegatedForAudience(gc, accessToken, parsers.GetHost(gc))
}

// ValidateDelegatedAccessTokenForResource is ValidateDelegatedAccessToken with
// the accepted audience supplied by the caller instead of derived from the
// request host. It exists for exactly one caller — ValidateMCPAccessToken — and
// the separation is the security boundary, not a convenience.
//
// # Why a second entry point rather than a looser audience rule
//
// ValidateDelegatedAccessToken has ONE caller,
// GetUserIDFromSessionOrAccessToken, which is the default rule behind /graphql,
// /v1/* and gRPC. Widening the audience check INSIDE it — accepting
// "<url>/mcp" alongside "<url>" — would therefore make an MCP-bound delegated
// token valid on every first-party surface as a side effect. That is precisely
// the audience confusion firstPartyAudienceOK exists to prevent, and it would
// be asymmetric: the stateful path rejects every resource-bound audience while
// the delegated path silently accepted one.
//
// # The invariant this preserves
//
//	f(aud) = surface, and it is a BIJECTION.
//
// Every token is valid at exactly one surface: the one its audience names. The
// match here is therefore EXACT (sameAudience), never "hostname or resource" —
// accepting both would break the bijection from the other side, letting a
// delegated token minted for the first-party API authenticate at /mcp too.
//
// Fails closed on an empty resource for the same reason ValidateMCPAccessToken
// does: an empty expected audience must never degrade into "accept anything".
func (p *provider) ValidateDelegatedAccessTokenForResource(gc *gin.Context, accessToken string, resource string) (map[string]interface{}, error) {
	if strings.TrimSpace(resource) == "" {
		return map[string]interface{}{}, fmt.Errorf(`unauthorized: no resource configured`)
	}
	return p.validateDelegatedForAudience(gc, accessToken, resource)
}

// validateDelegatedForAudience is the shared core. expectedAud is the ONLY thing
// that varies between the first-party and MCP entry points; every other check —
// the act claim, delegation-session liveness, subject liveness, issuer, subject
// and token type — is identical by construction, so a fix to any of them cannot
// land on one surface and miss the other.
//
// Note that expectedAud is deliberately NOT used as the issuer. The `iss` claim
// is always this server's bare URL regardless of which resource the token is
// bound to, so the issuer check below keeps using parsers.GetHost. Conflating
// the two would make an MCP token's issuer "<url>/mcp", which no token this
// server mints ever carries.
func (p *provider) validateDelegatedForAudience(gc *gin.Context, accessToken string, expectedAud string) (map[string]interface{}, error) {
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
	// ABSOLUTE URI and stamps it verbatim as `aud`, so the only way to reach a
	// surface here is to have named that surface's resource at exchange time:
	// this server's own URL for the first-party API, "<url>/mcp" for the MCP
	// transport. The audience must equal the caller-supplied expectedAud — not
	// the opaque --client-id, which no resource indicator can ever be.
	//
	// EXACT, never "one of". Each entry point passes exactly one expectedAud, so
	// a token minted for the first-party API does not authenticate at /mcp and
	// an MCP-bound token does not authenticate at /graphql. Relaxing this to
	// accept both audiences is the one change that would collapse the bijection
	// described on ValidateDelegatedAccessTokenForResource.
	//
	// Getting this wrong in the strict direction is not a safe failure: it
	// makes the delegated path unreachable by any token this deployment can
	// mint, so the feature silently does nothing. Getting it wrong in the loose
	// direction accepts a token minted for a different resource server, which
	// is audience confusion. Both are tested end to end through /oauth/token.
	//
	// Compared after trimming a trailing slash so "https://auth.example.com"
	// and "https://auth.example.com/" are the same audience — otherwise the
	// caller's exact spelling of the resource decides whether auth works.
	aud, _ := res["aud"].(string)
	if !sameAudience(aud, expectedAud) {
		if u, uErr := url.Parse(aud); uErr == nil && u.IsAbs() {
			p.dependencies.Log.Debug().Str("aud", aud).Str("expected", expectedAud).
				Msg("delegated token rejected: audience names a different resource server")
		}
		return res, fmt.Errorf(`unauthorized: token audience is not this server`)
	}

	// Session first, subject second: the session lookup hits the memory store
	// (in-process or Redis) while the subject lookup hits the database, so a
	// token whose delegation has already been revoked is rejected without
	// spending a DB read.
	if !p.delegationSessionIsLive(res) {
		return res, fmt.Errorf(`unauthorized: originating session is no longer valid`)
	}

	if !p.subjectIsLive(gc, userID) {
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

// subjectIsLive reports whether the subject a token was minted for is still
// active. It is the subject-liveness rule for EVERY stateful token check:
// validateStatefulAccessToken (so GraphQL, gRPC, REST and MCP alike) and RFC
// 8693 delegation.
//
// The subject is NOT always a user. A client_credentials token's `sub` is the
// service account's surrogate id, and RFC 8693 token exchange also accepts a
// service account as the subject, which is how a multi-hop chain is expressed
// (agent A delegates to agent B). userIsRevoked only ever looked the subject up
// as a USER and treated "not found" as "not revoked", so for a service-account
// subject it reported the caller live and the token kept working for its full
// lifetime after the account had been deactivated — deactivation stopped new
// tokens being issued but not the ones already out, and stopped nothing at all
// in a chain it had seeded.
//
// Resolution order mirrors how the token endpoint validates the subject at
// mint time (see handleTokenExchangeGrant): try user, then client.
//
// Fails CLOSED when the subject resolves to neither, and also when the store
// could not answer: a delegated token is stateless and short-lived, so a subject
// we cannot confirm is live must not authenticate — the same rule the exchange
// applies before it will seed a delegation at all. First-party callers make the
// opposite choice about an unanswerable lookup; see subjectLiveness.
func (p *provider) subjectIsLive(gc *gin.Context, subject string) bool {
	live, _ := p.subjectLiveness(gc, subject)
	return live
}

// subjectLiveness reports whether a token's subject is still active AND whether
// that could be determined at all:
//
//	live=true               — confirmed active
//	live=false, known=true  — confirmed absent, revoked, or deactivated
//	known=false             — the store could not answer (outage, timeout)
//
// The three-way answer exists because "the row is not there" and "the query
// failed" are different outcomes, and collapsing them is a mistake this codebase
// has already paid for — see AGENTS.md's not-found contract, and the note this
// function inherits from userIsRevoked, which it replaced: a lookup failure must
// never block a request that otherwise validated, "so a transient storage blip
// can't take down every authenticated request".
//
// That matters far more now than when only delegation consulted it. This is the
// liveness rule for every stateful access token and browser session, so treating
// an unreachable database as "subject not active" would 401 every authenticated
// request on GraphQL, gRPC and REST at once — and a 401 tells the SDKs the
// session expired, so a five-second failover becomes a fleet-wide forced logout
// reported to operators as a bad credential rather than an outage.
//
// So callers decide what an unknown means. First-party traffic tolerates it and
// lets the handler's own storage call surface the outage as a 500; the stateless
// delegated path fails closed (subjectIsLive). What neither tolerates is a
// CONFIRMED dead subject, which is the whole point: the subject is not always a
// user — a client_credentials token's `sub` is the service account's surrogate
// id, and RFC 8693 exchange accepts a service account as the subject too (agent A
// delegating to agent B). Resolving as a user only, and reading "no such user" as
// "not revoked", meant deactivating a service account did nothing to the tokens
// already issued to it.
//
// Resolution order mirrors how the token endpoint validates the subject at mint
// time (see handleTokenExchangeGrant): try user, then client.
func (p *provider) subjectLiveness(gc *gin.Context, subject string) (live, known bool) {
	if subject == "" {
		return false, true
	}
	if p.dependencies.StorageProvider == nil {
		// No store to consult is not evidence of anything.
		return false, false
	}

	user, userErr := p.dependencies.StorageProvider.GetUserByID(gc, subject)
	if userErr == nil && user != nil {
		if user.RevokedTimestamp != nil {
			p.dependencies.Log.Debug().Str("subject", subject).
				Msg("token rejected: subject user is revoked")
			return false, true
		}
		return true, true
	}

	// The client lookup is attempted whenever the user lookup did not POSITIVELY
	// find a user — never gated on the user error being a recognisable
	// not-found.
	//
	// That distinction is the whole correctness of this function across
	// backends. A machine token's subject is a client row id, so the user lookup
	// always misses; if a miss on some backend produced an unrecognised error
	// and short-circuited here, the client lookup would never run. On DynamoDB
	// GetUserByID returns a bare errors.New("no documets found") and on
	// Couchbase a gocb.ErrNoResult from the query path — neither satisfies
	// storage.IsNotFound. Gating on it therefore broke both directions at once
	// on exactly those two backends: delegated tokens with service-account
	// subjects were rejected outright, and deactivating a service account
	// stopped revoking its live tokens. CI runs SQLite only, so nothing failed.
	client, clientErr := p.dependencies.StorageProvider.GetClientByID(gc, subject)
	if clientErr == nil && client != nil {
		if !client.IsActive {
			p.dependencies.Log.Debug().Str("subject", subject).
				Msg("token rejected: subject service account is deactivated")
			return false, true
		}
		return true, true
	}

	// Neither table resolved the subject. Absence is only CONFIRMED when BOTH
	// lookups said "no such row"; if either was merely inconclusive the honest
	// answer is that we do not know, and callers choose what that means.
	if storage.IsNotFound(userErr) && storage.IsNotFound(clientErr) {
		p.dependencies.Log.Debug().Str("subject", subject).
			Msg("token rejected: subject resolves to neither an active user nor an active client")
		return false, true
	}

	p.dependencies.Log.Debug().AnErr("user_lookup", userErr).AnErr("client_lookup", clientErr).
		Str("subject", subject).Msg("subject liveness undetermined")
	return false, false
}
