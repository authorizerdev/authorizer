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
		badTokenExchangeRequest(gc, "subject_token and subject_token_type are required")
		return
	}
	if !isSupportedExchangeTokenType(subjectTokenType) {
		badTokenExchangeRequest(gc, "unsupported subject_token_type")
		return
	}

	// Delegation-only (DC1 / P3): an actor_token MUST be present. A subject-only
	// exchange is impersonation — a separate, admin-gated design not served on
	// this endpoint. Reject fail-closed rather than silently impersonating.
	if actorToken == "" {
		badTokenExchangeRequest(gc, "actor_token is required: only the delegation profile is supported here (impersonation is not permitted)")
		return
	}
	if actorTokenType == "" || !isSupportedExchangeTokenType(actorTokenType) {
		badTokenExchangeRequest(gc, "unsupported or missing actor_token_type")
		return
	}

	// RFC 8707 (DC4): EXACTLY ONE resource must bind the exchanged token. Read the
	// raw repeated form values so both 0 and >1 are rejected — a multi-audience
	// delegated token would be replayable across resource servers. Scoped to the
	// token-exchange path only; other grants are unaffected.
	resources := gc.PostFormArray("resource")
	if len(resources) != 1 || strings.TrimSpace(resources[0]) == "" {
		badTokenExchangeRequest(gc, "exactly one resource parameter is required")
		return
	}
	resource := strings.TrimSpace(resources[0])

	hostname := parsers.GetHost(gc)

	// Validate both tokens are valid, unexpired, Authorizer-issued access tokens
	// (signature + exp via ParseJWTToken, iss bound to this host, token_type).
	subjectClaims, err := h.validateExchangeToken(subjectToken, hostname)
	if err != nil {
		log.Debug().Err(err).Msg("invalid subject_token")
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_grant",
			"error_description": "The subject_token is invalid or has expired",
		})
		return
	}
	actorClaims, err := h.validateExchangeToken(actorToken, hostname)
	if err != nil {
		log.Debug().Err(err).Msg("invalid actor_token")
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_grant",
			"error_description": "The actor_token is invalid or has expired",
		})
		return
	}
	// RFC 8693 §1.1: the actor_token represents the acting party. Bind it to the
	// authenticated agent — a valid-but-unrelated token must not stand in as the
	// actor. The agent's own machine token (client_credentials) carries
	// sub = the service-account's surrogate ID (see token.go ServiceAccountID: sa.ID),
	// so require the actor_token's subject to be this client.
	if actorSub, _ := actorClaims["sub"].(string); actorSub != agent.ID {
		log.Debug().Msg("actor_token does not belong to the authenticated client")
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_grant",
			"error_description": "The actor_token must belong to the authenticated client",
		})
		return
	}

	subject, _ := subjectClaims["sub"].(string)
	if subject == "" {
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_grant",
			"error_description": "The subject_token has no subject",
		})
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
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_grant",
				"error_description": "The subject could not be verified",
			})
			return
		}
		if !subjectClient.IsActive {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_grant",
				"error_description": "The subject is no longer active",
			})
			return
		}
	} else {
		user, uErr := h.StorageProvider.GetUserByID(gc, subject)
		if uErr != nil || user == nil {
			// A user authority we cannot load must not seed a delegation (fail closed).
			log.Debug().Err(uErr).Msg("subject user could not be verified")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_grant",
				"error_description": "The subject could not be verified",
			})
			return
		}
		if user.RevokedTimestamp != nil {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_grant",
				"error_description": "The subject is no longer active",
			})
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
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_scope",
			"error_description": "The agent has no authorized scopes",
		})
		return
	}
	effective := intersectScopes(subjectScope, ceiling)
	if requested := strings.Fields(scopeParam); len(requested) > 0 {
		effective = intersectScopes(effective, requested)
	}
	if len(effective) == 0 {
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_scope",
			"error_description": "The requested scope is empty after attenuation against the subject and the agent ceiling",
		})
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
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "The delegation chain exceeds the maximum allowed depth",
		})
		return
	}

	delegated, err := h.TokenProvider.CreateDelegatedAccessToken(&token.DelegationTokenConfig{
		Subject:  subject,
		Actor:    act,
		Audience: resource,
		Scope:    effective,
		ClientID: agent.ClientID,
		HostName: hostname,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to mint delegated token")
		gc.JSON(http.StatusInternalServerError, gin.H{
			"error":             "server_error",
			"error_description": "Could not complete token issuance",
		})
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

	// RFC 8693 §2.2 token-exchange response.
	gc.JSON(http.StatusOK, gin.H{
		"access_token":      delegated.Token,
		"issued_token_type": constants.TokenTypeURNAccessToken,
		"token_type":        "Bearer",
		"expires_in":        expiresIn,
		"scope":             strings.Join(effective, " "),
	})
}

// badTokenExchangeRequest writes the RFC 6749 §5.2 invalid_request response.
func badTokenExchangeRequest(gc *gin.Context, desc string) {
	gc.JSON(http.StatusBadRequest, gin.H{
		"error":             "invalid_request",
		"error_description": desc,
	})
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
