package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RFC 8693 token exchange used to verify a subject_token's signature, issuer and
// token_type and nothing else. delegationSessionIsLive then refused the RESULTING
// token at Authorizer's own surfaces if the originating session had gone — but a
// delegated token is bound to a third-party `resource` and is validated by that
// server, which has no view of this session store.
//
// So a user could log out and their agent would keep minting fresh, externally
// valid credentials on their behalf until the subject_token expired, with nothing
// downstream able to tell. These tests pin the mint-time check and, just as
// importantly, the two cases it must NOT touch.

// TestTokenExchangeRejectsAfterSubjectLogout is the regression test.
func TestTokenExchangeRejectsAfterSubjectLogout(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
	subjectToken := testAccessToken(t, ts)
	actor := agentAccessToken(t, ts, router, clientID, secret)

	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", accessTokenType)
	form.Set("actor_token", actor)
	form.Set("actor_token_type", accessTokenType)
	form.Set("resource", "https://api.example.com/v1")

	// Baseline: the exchange works while the user's session is live.
	w := postTokenExchange(ts, router, form, clientID, secret)
	require.Equal(t, http.StatusOK, w.Code, "baseline delegation must succeed: %s", w.Body.String())

	userID, _ := decodeJWTPayload(t, subjectToken)["sub"].(string)
	require.NotEmpty(t, userID)

	// Log the user out — the same store delete logout, password reset and admin
	// revoke all perform.
	require.NoError(t, ts.MemoryStoreProvider.DeleteAllUserSessions(userID))

	w = postTokenExchange(ts, router, form, clientID, secret)
	require.Equal(t, http.StatusBadRequest, w.Code,
		"a logged-out subject must not seed a NEW delegation: %s", w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid_grant", resp["error"])
	// Opaque on purpose: identical to the invalid-subject_token response, so the
	// endpoint cannot be used to probe whether a given user is currently signed in.
	assert.Equal(t, "The subject_token is invalid or has expired", resp["error_description"])
}

// TestTokenExchangeServiceAccountSubjectIsExemptFromSessionCheck pins the
// exemption rather than leaving it incidental.
//
// A client_credentials token has no browser session, so a session check applied
// to it would reject every agent-to-agent hop — the multi-hop chain
// TestTokenExchangeMultiHopDelegation covers. Liveness for a service-account
// subject is established by the IsActive lookup instead, which
// TestDelegatedTokenForDeactivatedServiceAccountIsRejected covers.
func TestTokenExchangeServiceAccountSubjectIsExemptFromSessionCheck(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	// The SUBJECT is an agent's own machine token — no user, no session anywhere.
	subjectAgentID, subjectAgentSecret := newDelegationAgent(t, ts, "openid,profile")
	subjectToken := agentAccessToken(t, ts, router, subjectAgentID, subjectAgentSecret)

	callerID, callerSecret := newDelegationAgent(t, ts, "openid,profile")
	actor := agentAccessToken(t, ts, router, callerID, callerSecret)

	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", accessTokenType)
	form.Set("actor_token", actor)
	form.Set("actor_token_type", accessTokenType)
	form.Set("resource", "https://api.example.com/v1")

	w := postTokenExchange(ts, router, form, callerID, callerSecret)
	assert.Equal(t, http.StatusOK, w.Code,
		"a service-account subject has no session and must stay exempt: %s", w.Body.String())
}

// TestTokenExchangeChainedHopFollowsTheSubjectSession pins that the check reads
// the `sid` a chained exchange carries, not just a first-hop `nonce`.
//
// A delegated token deliberately carries no login_method or nonce claim, so hop 2
// resolves its session through the `sid` hop 1 stamped. If the check only ever
// looked at `nonce`, hop 2 would silently skip it and a logout would stop the
// first hop while leaving every subsequent one working.
func TestTokenExchangeChainedHopFollowsTheSubjectSession(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	agent1ID, agent1Secret := newDelegationAgent(t, ts, "openid,email,profile")
	agent2ID, agent2Secret := newDelegationAgent(t, ts, "openid,email")

	exchange := func(subjectToken, agentID, agentSecret string) *http.Response {
		actor := agentAccessToken(t, ts, router, agentID, agentSecret)
		form := url.Values{}
		form.Set("grant_type", tokenExchangeGrant)
		form.Set("subject_token", subjectToken)
		form.Set("subject_token_type", accessTokenType)
		form.Set("actor_token", actor)
		form.Set("actor_token_type", accessTokenType)
		form.Set("resource", "https://api.example.com/v1")
		return postTokenExchange(ts, router, form, agentID, agentSecret).Result()
	}

	userToken := testAccessToken(t, ts)
	userID, _ := decodeJWTPayload(t, userToken)["sub"].(string)
	require.NotEmpty(t, userID)

	// Hop 1 while the session is live.
	res1 := exchange(userToken, agent1ID, agent1Secret)
	require.Equal(t, http.StatusOK, res1.StatusCode, "hop 1 must succeed")
	var resp1 map[string]interface{}
	require.NoError(t, json.NewDecoder(res1.Body).Decode(&resp1))
	hop1Token, _ := resp1["access_token"].(string)
	require.NotEmpty(t, hop1Token)

	// Hop 2 from the hop-1 token, still live.
	res2 := exchange(hop1Token, agent2ID, agent2Secret)
	require.Equal(t, http.StatusOK, res2.StatusCode, "hop 2 must succeed while the session is live")

	// Now log the user out and retry hop 2 with the same hop-1 token.
	require.NoError(t, ts.MemoryStoreProvider.DeleteAllUserSessions(userID))

	res3 := exchange(hop1Token, agent2ID, agent2Secret)
	assert.Equal(t, http.StatusBadRequest, res3.StatusCode,
		"a chained hop must resolve the session through `sid` and stop at logout too")
}
