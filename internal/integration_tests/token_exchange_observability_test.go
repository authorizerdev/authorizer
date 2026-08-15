package integration_tests

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// An agent's activity has to be attributable — that is the entire point of the
// RFC 8693 act chain. Success was audited and counted; every one of the fourteen
// refusal paths was silent, with no audit row and no metric, while the sibling
// client_credentials grant already audited its failures. An agent probing the
// delegation endpoint therefore left no trail at all.

func authEventCount(event, status string) float64 {
	return testutil.ToFloat64(metrics.AuthEventsTotal.WithLabelValues(event, status))
}

func securityEventCount(event, reason string) float64 {
	return testutil.ToFloat64(metrics.SecurityEventsTotal.WithLabelValues(event, reason))
}

// TestTokenExchangeRejectionIsAuditedAndMetered covers the rejection path end to
// end: audit row, delegation-specific failure counter, and a reason label that
// names the rule which refused.
func TestTokenExchangeRejectionIsAuditedAndMetered(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
	agent, err := ts.StorageProvider.GetClientByClientID(ctx, clientID)
	require.NoError(t, err)
	require.NotNil(t, agent)

	beforeFailures := authEventCount(metrics.EventTokenExchange, metrics.StatusFailure)
	beforeReason := securityEventCount("token_exchange_rejected", "missing_actor_token")

	// A subject-only exchange: impersonation, which this profile refuses.
	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", testAccessToken(t, ts))
	form.Set("subject_token_type", accessTokenType)
	form.Set("resource", "https://api.example.com/v1")
	w := postTokenExchange(ts, router, form, clientID, secret)
	require.Equal(t, http.StatusBadRequest, w.Code)

	assert.Equal(t, beforeFailures+1, authEventCount(metrics.EventTokenExchange, metrics.StatusFailure),
		"a refused exchange must be counted, or there is no failure rate to alert on")
	assert.Equal(t, beforeReason+1, securityEventCount("token_exchange_rejected", "missing_actor_token"),
		"the reason label must name the rule that refused, so a misconfigured agent is "+
			"distinguishable from one probing the endpoint")

	// Audit writes are fire-and-forget (asyncutil.Go), so poll.
	var logs []*schemas.AuditLog
	require.Eventually(t, func() bool {
		var lErr error
		logs, _, lErr = ts.StorageProvider.ListAuditLogs(ctx, &model.Pagination{Limit: 50, Page: 1},
			map[string]interface{}{"action": constants.AuditTokenExchangeFailedEvent})
		return lErr == nil && len(logs) > 0
	}, 5*time.Second, 25*time.Millisecond,
		"a refused delegation must leave an audit trail; the client_credentials grant already does")

	var found bool
	for _, l := range logs {
		if l.ActorID != agent.ID {
			continue
		}
		found = true
		assert.Equal(t, constants.AuditActorTypeServiceAccount, l.ActorType)
		assert.Equal(t, "missing_actor_token", l.Metadata,
			"the audit row records the rule that refused")
	}
	require.True(t, found, "no rejection audit entry attributed to the calling agent")
}

// TestTokenExchangeSuccessIsMetered pins the denominator. A failure count with no
// success count beside it cannot express a rate, and the generic
// EventTokenIssued counter carries no grant label — delegated issuance was
// indistinguishable from authorization_code or client_credentials.
func TestTokenExchangeSuccessIsMetered(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", testAccessToken(t, ts))
	form.Set("subject_token_type", accessTokenType)
	form.Set("actor_token", agentAccessToken(t, ts, router, clientID, secret))
	form.Set("actor_token_type", accessTokenType)
	form.Set("resource", "https://api.example.com/v1")

	before := authEventCount(metrics.EventTokenExchange, metrics.StatusSuccess)
	w := postTokenExchange(ts, router, form, clientID, secret)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	assert.Equal(t, before+1, authEventCount(metrics.EventTokenExchange, metrics.StatusSuccess),
		"delegated issuance must be separately countable from every other grant")
}

// TestTokenExchangeSessionRejectionCarriesItsOwnReason pins that the audit
// record distinguishes a logged-out subject from a malformed one even though the
// HTTP response deliberately does not.
//
// The response is intentionally the same opaque invalid_grant, so the endpoint
// cannot be used to probe who is signed in. That opacity is the right call for
// the caller and the wrong one for the operator — the audit row is written
// server-side and never reaches the agent, so it can and must be specific.
func TestTokenExchangeSessionRejectionCarriesItsOwnReason(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
	subjectToken := testAccessToken(t, ts)
	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", accessTokenType)
	form.Set("actor_token", agentAccessToken(t, ts, router, clientID, secret))
	form.Set("actor_token_type", accessTokenType)
	form.Set("resource", "https://api.example.com/v1")

	userID, _ := decodeJWTPayload(t, subjectToken)["sub"].(string)
	require.NotEmpty(t, userID)
	require.NoError(t, ts.MemoryStoreProvider.DeleteAllUserSessions(userID))

	before := securityEventCount("token_exchange_rejected", "subject_session_not_live")
	w := postTokenExchange(ts, router, form, clientID, secret)
	require.Equal(t, http.StatusBadRequest, w.Code)

	assert.Equal(t, before+1,
		securityEventCount("token_exchange_rejected", "subject_session_not_live"),
		"a logout-driven refusal must be distinguishable in telemetry from a malformed token, "+
			"even though both return the same opaque invalid_grant to the caller")
}

// TestDelegatedInsufficientScopeIsMetered covers the agent scope-ceiling
// enforcement point, which was silent on every transport. Without it an operator
// has no way to see agents hitting their ceiling — the number needed before
// deciding whether widening one is justified.
func TestDelegatedInsufficientScopeIsMetered(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	delegated, _, _ := mintDelegatedViaEndpoint(t, ts, tokenRouter, testAuthorizerHost(ts))

	router := setupTestRouter(ts)
	post := func(query string) string {
		body := `{"query":` + jsonQuote(query) + `}`
		w := sendTestRequest(t, router, "POST", "/graphql", body, map[string]string{
			"Content-Type":     "application/json",
			"Authorization":    "Bearer " + delegated,
			"Origin":           "http://localhost:3000",
			"X-Authorizer-URL": testAuthorizerHost(ts),
		})
		return w.Body.String()
	}

	t.Run("an operation the agent lacks the scope for", func(t *testing.T) {
		before := securityEventCount("delegated_insufficient_scope", "graphql_scope_missing")
		out := post(`mutation { update_profile(params: {given_name: "mutated-by-agent"}) { message } }`)
		require.Contains(t, out, "insufficient_scope")
		assert.Equal(t, before+1,
			securityEventCount("delegated_insufficient_scope", "graphql_scope_missing"),
			"an agent refused for want of scope must be counted")
	})

	t.Run("an operation not delegatable at all", func(t *testing.T) {
		// Separate label: this one means the operation is not on the delegated
		// allow-list, which is a client bug or probing — a different action for
		// the operator than "grant this agent more scope".
		before := securityEventCount("delegated_insufficient_scope", "graphql_not_delegatable")
		out := post(`query { webauthn_credentials { id } }`)
		require.Contains(t, out, "insufficient_scope")
		assert.Equal(t, before+1,
			securityEventCount("delegated_insufficient_scope", "graphql_not_delegatable"),
			"a fail-closed refusal must be counted separately from a scope shortfall")
	})
}
