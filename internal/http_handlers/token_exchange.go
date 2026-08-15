package http_handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// maxActChainDepth caps the RFC 8693 `act` delegation nesting (design H1). A
// resulting chain deeper than this is rejected — an unbounded chain is both a
// token-bloat and an audit-legibility risk, and real delegation is a few hops
// (app > agent > sub-agent). Hard-coded rather than configurable: no operator
// need to loosen it has surfaced; promote to config if one ever does.
const maxActChainDepth = 4

// handleTokenExchangeGrant implements the RFC 8693 token-exchange grant in the
// DELEGATION-ONLY profile (design §3, DC1). The calling client (agent) has
// already been authenticated by the clientauth resolver as an active
// service_account; agent is that resolved client. It mints a short-lived,
// resource-bound (RFC 8707), attenuated access token that carries the nested
// `act` actor chain and whose `sub` is the subject_token's user. Impersonation
// (a subject-only exchange) is intentionally rejected here — it is a separate,
// admin-gated design.
func (h *httpProvider) handleTokenExchangeGrant(gc *gin.Context, agent *schemas.Client, reqBody *RequestBody, scopeParam string) {
	log := h.Log.With().Str("func", "handleTokenExchangeGrant").Logger()

	subjectToken := strings.TrimSpace(reqBody.SubjectToken)
	subjectTokenType := strings.TrimSpace(reqBody.SubjectTokenType)
	actorToken := strings.TrimSpace(reqBody.ActorToken)
	actorTokenType := strings.TrimSpace(reqBody.ActorTokenType)

	// RFC 8693 §2.1: subject_token and subject_token_type are REQUIRED.
	if subjectToken == "" || subjectTokenType == "" {
		h.badTokenExchangeRequest(gc, agent, reasonMissingSubject, "subject_token and subject_token_type are required")
		return
	}
	if !isSupportedExchangeTokenType(subjectTokenType) {
		h.badTokenExchangeRequest(gc, agent, reasonUnsupportedSubject, "unsupported subject_token_type")
		return
	}

	// Delegation-only (DC1 / P3): an actor_token MUST be present. A subject-only
	// exchange is impersonation — a separate, admin-gated design not served on
	// this endpoint. Reject fail-closed rather than silently impersonating.
	if actorToken == "" {
		h.badTokenExchangeRequest(gc, agent, reasonMissingActor, "actor_token is required: only the delegation profile is supported here (impersonation is not permitted)")
		return
	}
	if actorTokenType == "" || !isSupportedExchangeTokenType(actorTokenType) {
		h.badTokenExchangeRequest(gc, agent, reasonUnsupportedActor, "unsupported or missing actor_token_type")
		return
	}

	// RFC 8707 (DC4): EXACTLY ONE resource must bind the exchanged token. Read the
	// raw repeated form values so both 0 and >1 are rejected — a multi-audience
	// delegated token would be replayable across resource servers. Scoped to the
	// token-exchange path only; other grants are unaffected.
	resources := gc.PostFormArray("resource")
	if len(resources) != 1 || strings.TrimSpace(resources[0]) == "" {
		h.badTokenExchangeRequest(gc, agent, reasonResourceCount, "exactly one resource parameter is required")
		return
	}
	resource := strings.TrimSpace(resources[0])
	// RFC 8707 §2: the resource indicator MUST be an absolute URI without a
	// fragment. /authorize enforced this from the start; without the same check
	// here an agent could bind a delegated token's `aud` to an arbitrary opaque
	// string, which makes the audience restriction unenforceable at the resource
	// server. Same helper, same rejection, so the two paths cannot drift.
	if !isValidResourceIndicator(resource) {
		log.Debug().Str("client_id", agent.ClientID).Str("resource", resource).Msg("rejected: invalid resource indicator")
		h.rejectExchange(gc, agent, http.StatusBadRequest, reasonResourceInvalid,
			"invalid_target", "resource must be an absolute URI without a fragment")
		return
	}

	hostname := parsers.GetHost(gc)

	// Validate both tokens are valid, unexpired, Authorizer-issued access tokens
	// (signature + exp via ParseJWTToken, iss bound to this host, token_type).
	subjectClaims, err := h.validateExchangeToken(subjectToken, hostname)
	if err != nil {
		log.Debug().Err(err).Msg("invalid subject_token")
		h.rejectExchange(gc, agent, http.StatusBadRequest, reasonSubjectTokenInvalid,
			"invalid_grant", "The subject_token is invalid or has expired")
		return
	}
	actorClaims, err := h.validateExchangeToken(actorToken, hostname)
	if err != nil {
		log.Debug().Err(err).Msg("invalid actor_token")
		h.rejectExchange(gc, agent, http.StatusBadRequest, reasonActorTokenInvalid,
			"invalid_grant", "The actor_token is invalid or has expired")
		return
	}
	// RFC 8693 §1.1: the actor_token represents the acting party. Bind it to the
	// authenticated agent — a valid-but-unrelated token must not stand in as the
	// actor. The agent's own machine token (client_credentials) carries
	// sub = the service-account's surrogate ID (see token.go ServiceAccountID: sa.ID),
	// so require the actor_token's subject to be this client.
	if actorSub, _ := actorClaims["sub"].(string); actorSub != agent.ID {
		log.Debug().Msg("actor_token does not belong to the authenticated client")
		h.rejectExchange(gc, agent, http.StatusBadRequest, reasonActorNotCaller,
			"invalid_grant", "The actor_token must belong to the authenticated client")
		return
	}

	subject, _ := subjectClaims["sub"].(string)
	if subject == "" {
		h.rejectExchange(gc, agent, http.StatusBadRequest, reasonSubjectMissing,
			"invalid_grant", "The subject_token has no subject")
		return
	}

	// Fail-closed on a revoked authority source: a deprovisioned user (SCIM
	// active:false / RevokedTimestamp) must never seed a fresh delegation.
	// Distinguish a machine/service-account subject (a multi-hop agent chain, which
	// has no user row) from a user subject by the token's login_method — NOT by a
	// failed user lookup, so a transient DB error can't be silently treated as
	// "not a user" and fail open past the revocation check.
	onBehalfOfType := constants.AuditActorTypeUser
	if lm, _ := subjectClaims["login_method"].(string); lm == constants.AuthRecipeMethodServiceAccount {
		onBehalfOfType = "agent"
		// validateExchangeToken only checks the JWT's own signature/exp — it
		// has no session-store lookup, so a service-account subject_token
		// stays cryptographically valid until its natural expiry even after
		// the account is deactivated. Without this check, a still-unexpired
		// token from a just-deactivated service account could keep seeding
		// fresh delegated tokens through a willing downstream agent,
		// extending its effective lifetime past deactivation. Same
		// fail-closed contract as the user branch below: a subject we
		// cannot load or confirm active must not seed a delegation.
		subjectClient, cErr := h.StorageProvider.GetClientByID(gc, subject)
		if cErr != nil || subjectClient == nil {
			log.Debug().Err(cErr).Msg("subject service account could not be verified")
			h.rejectExchange(gc, agent, http.StatusBadRequest, reasonSubjectUnverifiable,
				"invalid_grant", "The subject could not be verified")
			return
		}
		if !subjectClient.IsActive {
			h.rejectExchange(gc, agent, http.StatusBadRequest, reasonSubjectInactive,
				"invalid_grant", "The subject is no longer active")
			return
		}
	} else {
		user, uErr := h.StorageProvider.GetUserByID(gc, subject)
		if uErr != nil || user == nil {
			// A user authority we cannot load must not seed a delegation (fail closed).
			log.Debug().Err(uErr).Msg("subject user could not be verified")
			h.rejectExchange(gc, agent, http.StatusBadRequest, reasonSubjectUnverifiable,
				"invalid_grant", "The subject could not be verified")
			return
		}
		if user.RevokedTimestamp != nil {
			h.rejectExchange(gc, agent, http.StatusBadRequest, reasonSubjectInactive,
				"invalid_grant", "The subject is no longer active")
			return
		}
	}

	// Attenuation (DC2/H1), fail-closed: effective = subject_scope ∩ agent_ceiling
	// (∩ requested when a scope was asked for). Monotonic non-widening: because we
	// ALWAYS intersect the subject_token's own scope, re-exchanging an
	// already-narrowed delegated token can only narrow further — never restore.
	subjectScope := claimToStringSlice(subjectClaims["scope"])
	ceiling := agent.ParsedAllowedScopes()
	if len(ceiling) == 0 {
		// Empty AllowedScopes is DENY-ALL (schema § AllowedScopes) — an agent with
		// no ceiling can delegate nothing.
		h.rejectExchange(gc, agent, http.StatusBadRequest, reasonAgentNoScopes,
			"invalid_scope", "The agent has no authorized scopes")
		return
	}
	effective := intersectScopes(subjectScope, ceiling)
	if requested := strings.Fields(scopeParam); len(requested) > 0 {
		effective = intersectScopes(effective, requested)
	}
	if len(effective) == 0 {
		h.rejectExchange(gc, agent, http.StatusBadRequest, reasonScopeEmpty,
			"invalid_scope", "The requested scope is empty after attenuation against the subject and the agent ceiling")
		return
	}

	// Nested `act` (DC3 / RFC 8693 §4.1): the immediate actor is the
	// AUTHENTICATED agent (its registered client_id — never a token-supplied
	// claim), and any prior act chain carried on the subject_token nests beneath
	// it, giving a multi-hop app > agent > sub-agent chain.
	act := map[string]interface{}{"sub": agent.ClientID}
	if prior, ok := subjectClaims["act"].(map[string]interface{}); ok && len(prior) > 0 {
		act["act"] = prior
	}
	if actChainDepth(act) > maxActChainDepth {
		h.rejectExchange(gc, agent, http.StatusBadRequest, reasonChainTooDeep,
			"invalid_request", "The delegation chain exceeds the maximum allowed depth")
		return
	}

	// Carry the subject's session forward so the delegation is revocable at
	// Authorizer's own API (token.DelegationSessionID). A chained exchange
	// re-exchanges an already-delegated token, which carries `sid` rather than
	// `nonce`, so propagate that verbatim — otherwise the second hop would lose
	// the binding and outlive the logout that killed the first.
	sessionID, _ := subjectClaims["sid"].(string)
	if sessionID == "" {
		nonce, _ := subjectClaims["nonce"].(string)
		loginMethod, _ := subjectClaims["login_method"].(string)
		sessionID = token.DelegationSessionID(loginMethod, subject, nonce)
	}

	// The delegation has to be revocable at MINT time, not only at use time.
	//
	// delegationSessionIsLive already refuses a delegated token whose originating
	// session is gone — but only at Authorizer's own surfaces. The token minted
	// here is bound to a third-party `resource` and is validated by THAT server,
	// which has no view of this session store. So without this check, a user who
	// logged out could still have their agent mint fresh credentials against
	// external resource servers for as long as the subject_token remained
	// unexpired, and nothing downstream would notice.
	//
	// Checked only when there is something to check. A `sid` is either well-formed
	// or absent — DelegationSessionID returns "" rather than a partial value — and
	// its absence is a documented state meaning "this subject had no session"
	// (see CreateDelegatedAccessToken, which omits the claim entirely in that
	// case, so its presence always means checkable). Rejecting that case too would
	// delete a supported branch rather than close the gap.
	//
	// Service-account subjects are exempt: a client_credentials token has no
	// browser session, and its liveness was already established by the IsActive
	// check above. Applying this to them would break the multi-hop agent chain.
	if onBehalfOfType == constants.AuditActorTypeUser && sessionID != "" {
		if sessionKey, sessionNonce, ok := token.ParseDelegationSessionID(sessionID); ok {
			if _, sErr := h.MemoryStoreProvider.GetUserSession(
				sessionKey, constants.TokenTypeAccessToken+"_"+sessionNonce); sErr != nil {
				log.Debug().Msg("rejected: the subject's originating session is no longer live")
				// Deliberately the same opaque invalid_grant the invalid-subject_token
				// path returns, so this is not an oracle for whether a given user is
				// currently signed in. The AUDIT record distinguishes them (the
				// reason constant differs) because that is written server-side and
				// never reaches the caller.
				h.rejectExchange(gc, agent, http.StatusBadRequest, reasonSubjectSessionGone,
					"invalid_grant", "The subject_token is invalid or has expired")
				return
			}
		}
	}

	delegated, err := h.TokenProvider.CreateDelegatedAccessToken(&token.DelegationTokenConfig{
		Subject:   subject,
		Actor:     act,
		Audience:  resource,
		Scope:     effective,
		ClientID:  agent.ClientID,
		HostName:  hostname,
		SessionID: sessionID,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to mint delegated token")
		h.rejectExchange(gc, agent, http.StatusInternalServerError, reasonMintFailed,
			"server_error", "Could not complete token issuance")
		return
	}

	expiresIn := delegated.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	// Audit the delegation chain (DC21). The audit schema has no dedicated
	// on-behalf-of columns yet; fold actor + on-behalf-of + chain into the JSON
	// Metadata column (queryable, zero multi-DB schema change). Promoting these to
	// first-class columns across all providers is deferred — see design §5.
	metadata, _ := json.Marshal(map[string]string{
		"grant_type":        constants.GrantTypeTokenExchange,
		"on_behalf_of":      subject,
		"on_behalf_of_type": onBehalfOfType,
		"delegation_chain":  delegationChainString(act, subject),
		"resource":          resource,
	})
	h.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditTokenIssuedEvent,
		ActorID:      agent.ID,
		ActorType:    constants.AuditActorTypeServiceAccount,
		ResourceType: constants.AuditResourceTypeToken,
		ResourceID:   subject,
		Metadata:     string(metadata),
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})

	metrics.RecordAuthEvent(metrics.EventTokenIssued, metrics.StatusSuccess)
	// Delegation-specific, alongside the generic issuance counter above. Without
	// it there is no denominator for the failure count rejectExchange records, and
	// no way to see delegated issuance apart from any other grant.
	metrics.RecordAuthEvent(metrics.EventTokenExchange, metrics.StatusSuccess)

	// RFC 8693 §2.2 token-exchange response.
	gc.JSON(http.StatusOK, gin.H{
		"access_token":      delegated.Token,
		"issued_token_type": constants.TokenTypeURNAccessToken,
		"token_type":        "Bearer",
		"expires_in":        expiresIn,
		"scope":             strings.Join(effective, " "),
	})
}

// Rejection reasons for rejectExchange. These become a Prometheus label and an
// audit Metadata value, so they are a FIXED, low-cardinality set — never a
// caller-derived string. Each names the rule that refused, which is the thing an
// operator needs in order to tell "this agent is misconfigured" from "this agent
// is probing".
const (
	reasonMissingSubject      = "missing_subject_token"
	reasonUnsupportedSubject  = "unsupported_subject_token_type"
	reasonMissingActor        = "missing_actor_token"
	reasonUnsupportedActor    = "unsupported_actor_token_type"
	reasonResourceCount       = "resource_not_exactly_one"
	reasonResourceInvalid     = "invalid_resource_indicator"
	reasonSubjectTokenInvalid = "subject_token_invalid"
	reasonActorTokenInvalid   = "actor_token_invalid"
	reasonActorNotCaller      = "actor_token_not_the_caller"
	reasonSubjectMissing      = "subject_token_has_no_subject"
	reasonSubjectUnverifiable = "subject_unverifiable"
	reasonSubjectInactive     = "subject_inactive"
	reasonSubjectSessionGone  = "subject_session_not_live"
	reasonAgentNoScopes       = "agent_has_no_scopes"
	reasonScopeEmpty          = "scope_empty_after_attenuation"
	reasonChainTooDeep        = "delegation_chain_too_deep"
	reasonMintFailed          = "token_issuance_failed"
)

// rejectExchange writes a token-exchange refusal AND records it.
//
// It exists because every one of the fourteen refusal paths on this endpoint was
// previously silent: no audit entry, no metric, only a Debug log. The
// client_credentials grant already audits its failures
// (AuditTokenClientCredentialsFailedEvent), so machine-identity auth failures
// were attributable while delegation failures — the more sensitive of the two,
// since a delegated token carries a user's authority — were not. An agent
// probing this endpoint left no trail.
//
// Routing every refusal through one function is also what keeps that true: a new
// rejection path added later cannot be silent without deliberately bypassing this.
func (h *httpProvider) rejectExchange(gc *gin.Context, agent *schemas.Client, status int, reason, errCode, desc string) {
	metrics.RecordAuthEvent(metrics.EventTokenExchange, metrics.StatusFailure)
	metrics.RecordSecurityEvent("token_exchange_rejected", reason)

	// agent is nil-safe: the caller is always an authenticated client by the time
	// this handler runs, but an audit call must never be the thing that panics an
	// auth endpoint.
	actorID := ""
	if agent != nil {
		actorID = agent.ID
	}
	h.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditTokenExchangeFailedEvent,
		ActorID:      actorID,
		ActorType:    constants.AuditActorTypeServiceAccount,
		ResourceType: constants.AuditResourceTypeToken,
		// The reason constant only — never the subject id or the token. A refusal
		// record must not become a place where an unverified subject's identity is
		// written on the strength of a request that was rejected.
		Metadata:  reason,
		IPAddress: utils.GetIP(gc.Request),
		UserAgent: utils.GetUserAgent(gc.Request),
	})

	gc.JSON(status, gin.H{
		"error":             errCode,
		"error_description": desc,
	})
}

// badTokenExchangeRequest writes the RFC 6749 §5.2 invalid_request response.
func (h *httpProvider) badTokenExchangeRequest(gc *gin.Context, agent *schemas.Client, reason, desc string) {
	h.rejectExchange(gc, agent, http.StatusBadRequest, reason, "invalid_request", desc)
}

// isSupportedExchangeTokenType reports whether an RFC 8693 subject/actor token
// type URN is one this delegation profile accepts (access token or generic JWT).
func isSupportedExchangeTokenType(t string) bool {
	return t == constants.TokenTypeURNAccessToken || t == constants.TokenTypeURNJWT
}

// validateExchangeToken verifies signature + exp (ParseJWTToken), binds iss to
// this host, and requires token_type=access_token. It deliberately does NOT
// require aud == this server's client_id: a first-party token's aud is the
// client_id, but a prior *delegated* token's aud is a resource URI (the
// sub-agent / multi-hop re-exchange case). Both are unforgeable — only this
// server's key signs them and iss binds them to this host — so signature + iss +
// token_type is the correct, non-widening trust gate for a token we issued.
func (h *httpProvider) validateExchangeToken(tokenStr, hostname string) (jwt.MapClaims, error) {
	claims, err := h.TokenProvider.ParseJWTToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if iss, _ := claims["iss"].(string); iss == "" || iss != hostname {
		return nil, errors.New("issuer mismatch")
	}
	if tt, _ := claims["token_type"].(string); tt != constants.TokenTypeAccessToken {
		return nil, errors.New("token is not an access token")
	}
	return claims, nil
}

// claimToStringSlice coerces a JWT `scope` claim (parsed as []interface{}) into
// a []string, dropping non-string / empty entries.
func claimToStringSlice(v interface{}) []string {
	switch vv := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(vv))
		for _, e := range vv {
			if s, ok := e.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return vv
	}
	return nil
}

// intersectScopes returns the scopes present in both a and b, preserving a's
// order and de-duplicating. This is the attenuation primitive.
func intersectScopes(a, b []string) []string {
	allowed := make(map[string]struct{}, len(b))
	for _, s := range b {
		allowed[s] = struct{}{}
	}
	out := make([]string, 0, len(a))
	seen := make(map[string]struct{}, len(a))
	for _, s := range a {
		if _, ok := allowed[s]; !ok {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// actChainDepth counts the nesting depth of an RFC 8693 `act` chain
// (1 for a single actor, +1 per nested act).
func actChainDepth(act map[string]interface{}) int {
	depth := 0
	for cur := act; cur != nil; {
		depth++
		next, _ := cur["act"].(map[string]interface{})
		cur = next
	}
	return depth
}

// delegationChainString renders the actor chain for the audit log in the
// design's inner→outer→subject order, e.g. "app:concierge>agent:bot>user:alice".
func delegationChainString(act map[string]interface{}, subject string) string {
	var actors []string
	for cur := act; cur != nil; {
		if s, _ := cur["sub"].(string); s != "" {
			actors = append(actors, s)
		}
		next, _ := cur["act"].(map[string]interface{})
		cur = next
	}
	parts := make([]string, 0, len(actors)+1)
	for i := len(actors) - 1; i >= 0; i-- {
		parts = append(parts, actors[i])
	}
	parts = append(parts, subject)
	return strings.Join(parts, ">")
}
