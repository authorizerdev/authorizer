package integration_tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const tokenExchangeGrant = "urn:ietf:params:oauth:grant-type:token-exchange"
const accessTokenType = "urn:ietf:params:oauth:token-type:access_token"

// decodeJWTPayload base64url-decodes a JWT's payload segment WITHOUT validating
// signature or audience. A delegated token's `aud` is the bound resource (not the
// deployment client_id), so the standard validator would reject it — the test
// inspects the claims directly.
func decodeJWTPayload(t *testing.T, tok string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(tok, ".")
	require.Len(t, parts, 3, "token must be a well-formed JWT")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &claims))
	return claims
}

// claimScope extracts a delegated/machine token's scope claim as []string.
// The claim is encoded as a JSON array (token.DelegationTokenConfig.Scope /
// AuthTokenConfig.Scope are both []string), so it decodes through
// encoding/json as []interface{}, NOT a space-separated string - a
// `claims["scope"].(string)` assertion fails silently (empty string, no
// panic), which previously made a widening regression on this exact claim
// undetectable by the test that specifically exists to catch one.
func claimScope(t *testing.T, claims map[string]interface{}) []string {
	t.Helper()
	raw, ok := claims["scope"].([]interface{})
	require.True(t, ok, "scope claim must be a JSON array")
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		require.True(t, ok, "scope claim entries must be strings")
		out = append(out, s)
	}
	return out
}

// newDelegationAgent creates an active service_account client (the agent) with the
// given scope ceiling, returning its client_id and plaintext secret.
func newDelegationAgent(t *testing.T, ts *testSetup, ceiling string) (string, string) {
	t.Helper()
	secret := "agent-secret-" + uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	require.NoError(t, err)
	agent, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
		Name:          "agent-" + uuid.New().String(),
		Kind:          constants.ClientKindServiceAccount,
		ClientSecret:  string(hash),
		AllowedScopes: ceiling,
		IsActive:      true,
	})
	require.NoError(t, err)
	return agent.ClientID, secret
}

// agentAccessToken mints the agent's OWN access token via client_credentials —
// this is the token an agent presents as its actor_token (its client_id claim
// binds it to the authenticated client per RFC 8693).
func agentAccessToken(t *testing.T, ts *testSetup, router http.Handler, clientID, secret string) string {
	t.Helper()
	f := url.Values{}
	f.Set("grant_type", "client_credentials")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(f.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
	req.SetBasicAuth(clientID, secret)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "agent client_credentials must succeed: %s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	tok, _ := resp["access_token"].(string)
	require.NotEmpty(t, tok)
	return tok
}

func postTokenExchange(ts *testSetup, router http.Handler, form url.Values, clientID, secret string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Match the host the minted subject/actor tokens use as their iss.
	req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
	req.SetBasicAuth(clientID, secret)
	router.ServeHTTP(w, req)
	return w
}

// TestTokenExchangeMultiHopDelegation exercises the act-chain nesting and
// scope attenuation the map/design docs describe (app > agent > sub-agent)
// through REAL multiple JWT round-trips - not just reading actChainDepth's
// logic, since a JWT's claims (including the nested `act` object) go
// through a JSON marshal/parse round-trip at every hop, exactly the kind of
// path where a nested-map type assertion could silently truncate the chain
// without a runtime error. Only single-hop delegation had test coverage
// before this.
func TestTokenExchangeMultiHopDelegation(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	// Each hop's agent has a progressively narrower scope ceiling, so a
	// widening bug would show up as a scope surviving past the hop that
	// should have dropped it, not just as an empty-vs-nonempty check. Scoped
	// to openid/email/profile - the standard OIDC scopes testAccessToken's
	// plain signup actually carries (attenuation only ever narrows what the
	// subject already has, so a ceiling of custom scopes the user's own
	// token never had would just intersect to empty at hop 1).
	agent1ID, agent1Secret := newDelegationAgent(t, ts, "openid,email,profile")
	agent2ID, agent2Secret := newDelegationAgent(t, ts, "openid,email")
	agent3ID, agent3Secret := newDelegationAgent(t, ts, "openid")
	agent4ID, agent4Secret := newDelegationAgent(t, ts, "openid")
	agent5ID, agent5Secret := newDelegationAgent(t, ts, "openid")

	exchange := func(t *testing.T, subjectToken, agentID, agentSecret string) *httptest.ResponseRecorder {
		t.Helper()
		actor := agentAccessToken(t, ts, router, agentID, agentSecret)
		form := url.Values{}
		form.Set("grant_type", tokenExchangeGrant)
		form.Set("subject_token", subjectToken)
		form.Set("subject_token_type", accessTokenType)
		form.Set("actor_token", actor)
		form.Set("actor_token_type", accessTokenType)
		form.Set("resource", "https://api.example.com/v1")
		return postTokenExchange(ts, router, form, agentID, agentSecret)
	}

	userToken := testAccessToken(t, ts)

	w1 := exchange(t, userToken, agent1ID, agent1Secret)
	require.Equal(t, http.StatusOK, w1.Code, "hop 1: %s", w1.Body.String())
	var resp1 map[string]interface{}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &resp1))
	tok1, _ := resp1["access_token"].(string)
	claims1 := decodeJWTPayload(t, tok1)
	assert.ElementsMatch(t, []string{"openid", "email", "profile"}, claimScope(t, claims1), "hop 1 scope = agent1 ceiling ∩ user's own scope")
	act1, ok := claims1["act"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, agent1ID, act1["sub"])
	assert.Nil(t, act1["act"], "hop 1 has no prior actor")

	w2 := exchange(t, tok1, agent2ID, agent2Secret)
	require.Equal(t, http.StatusOK, w2.Code, "hop 2: %s", w2.Body.String())
	var resp2 map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	tok2, _ := resp2["access_token"].(string)
	claims2 := decodeJWTPayload(t, tok2)
	assert.ElementsMatch(t, []string{"openid", "email"}, claimScope(t, claims2), "hop 2 narrows to agent2's ceiling (profile dropped)")
	act2, ok := claims2["act"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, agent2ID, act2["sub"], "immediate actor is the hop-2 agent")
	act2prior, ok := act2["act"].(map[string]interface{})
	require.True(t, ok, "hop 1's actor must survive nested under hop 2 through the real JWT round-trip")
	assert.Equal(t, agent1ID, act2prior["sub"], "hop 1's agent is preserved as the nested prior actor")

	w3 := exchange(t, tok2, agent3ID, agent3Secret)
	require.Equal(t, http.StatusOK, w3.Code, "hop 3: %s", w3.Body.String())
	var resp3 map[string]interface{}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &resp3))
	tok3, _ := resp3["access_token"].(string)
	claims3 := decodeJWTPayload(t, tok3)
	assert.ElementsMatch(t, []string{"openid"}, claimScope(t, claims3), "hop 3 narrows further to agent3's ceiling (email dropped)")

	w4 := exchange(t, tok3, agent4ID, agent4Secret)
	require.Equal(t, http.StatusOK, w4.Code, "hop 4 (at the depth cap) must succeed: %s", w4.Body.String())
	var resp4 map[string]interface{}
	require.NoError(t, json.Unmarshal(w4.Body.Bytes(), &resp4))
	tok4, _ := resp4["access_token"].(string)
	claims4 := decodeJWTPayload(t, tok4)
	// Walk the full 4-level nested act chain end to end - proves depth
	// counting and nesting both survived four real JWT round-trips intact.
	act4, ok := claims4["act"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, agent4ID, act4["sub"])
	act4p1, ok := act4["act"].(map[string]interface{})
	require.True(t, ok, "depth 2 missing")
	assert.Equal(t, agent3ID, act4p1["sub"])
	act4p2, ok := act4p1["act"].(map[string]interface{})
	require.True(t, ok, "depth 3 missing")
	assert.Equal(t, agent2ID, act4p2["sub"])
	act4p3, ok := act4p2["act"].(map[string]interface{})
	require.True(t, ok, "depth 4 missing")
	assert.Equal(t, agent1ID, act4p3["sub"])
	assert.Nil(t, act4p3["act"], "chain must be exactly 4 deep, no further nesting")

	w5 := exchange(t, tok4, agent5ID, agent5Secret)
	assert.Equal(t, http.StatusBadRequest, w5.Code, "hop 5 exceeds maxActChainDepth and must be rejected: %s", w5.Body.String())
	assert.Contains(t, w5.Body.String(), "maximum allowed depth")
}

func TestTokenExchangeDelegation(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	baseForm := func(agentScopeCeiling string) (url.Values, string, string) {
		clientID, secret := newDelegationAgent(t, ts, agentScopeCeiling)
		subject := testAccessToken(t, ts)
		// actor_token is the agent's OWN token (client_id bound to the agent).
		actor := agentAccessToken(t, ts, router, clientID, secret)
		form := url.Values{}
		form.Set("grant_type", tokenExchangeGrant)
		form.Set("subject_token", subject)
		form.Set("subject_token_type", accessTokenType)
		form.Set("actor_token", actor)
		form.Set("actor_token_type", accessTokenType)
		form.Set("resource", "https://api.example.com/v1")
		return form, clientID, secret
	}

	t.Run("actor_token_required_delegation_only", func(t *testing.T) {
		clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
		form := url.Values{}
		form.Set("grant_type", tokenExchangeGrant)
		form.Set("subject_token", "any.subject.token")
		form.Set("subject_token_type", accessTokenType)
		form.Set("resource", "https://api.example.com/v1")
		// no actor_token
		w := postTokenExchange(ts, router, form, clientID, secret)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "actor_token", "impersonation (subject-only) must be rejected")
	})

	t.Run("exactly_one_resource_required", func(t *testing.T) {
		clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
		mk := func() url.Values {
			f := url.Values{}
			f.Set("grant_type", tokenExchangeGrant)
			f.Set("subject_token", "any.subject.token")
			f.Set("subject_token_type", accessTokenType)
			f.Set("actor_token", "any.actor.token")
			f.Set("actor_token_type", accessTokenType)
			return f
		}
		// zero resources
		w := postTokenExchange(ts, router, mk(), clientID, secret)
		assert.Equal(t, http.StatusBadRequest, w.Code, "no resource must be rejected")
		// two resources
		f := mk()
		f.Add("resource", "https://a.example.com")
		f.Add("resource", "https://b.example.com")
		w = postTokenExchange(ts, router, f, clientID, secret)
		assert.Equal(t, http.StatusBadRequest, w.Code, "multiple resources must be rejected")
	})

	t.Run("happy_path_nested_act_and_resource_aud", func(t *testing.T) {
		form, clientID, secret := baseForm("openid,profile,email,read")
		form.Set("scope", "profile email")
		w := postTokenExchange(ts, router, form, clientID, secret)
		require.Equal(t, http.StatusOK, w.Code, "valid delegation must issue a token: %s", w.Body.String())
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		tok, _ := resp["access_token"].(string)
		require.NotEmpty(t, tok)
		claims := decodeJWTPayload(t, tok)
		// aud is the single bound resource (RFC 8707).
		assert.Equal(t, "https://api.example.com/v1", claims["aud"])
		// sub is the user (authority source), not the agent.
		assert.NotEmpty(t, claims["sub"])
		// act carries the immediate actor = the calling agent's client_id (DC3).
		act, ok := claims["act"].(map[string]interface{})
		require.True(t, ok, "issued token must carry an act claim")
		assert.Equal(t, clientID, act["sub"], "act.sub must be the delegating agent")
	})

	t.Run("attenuation_cannot_widen_beyond_agent_ceiling", func(t *testing.T) {
		// Agent ceiling excludes "admin"; requesting it must not grant it.
		form, clientID, secret := baseForm("openid,profile,email")
		_ = clientID
		form.Set("scope", "profile admin")
		w := postTokenExchange(ts, router, form, clientID, secret)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		tok, _ := resp["access_token"].(string)
		require.NotEmpty(t, tok)
		claims := decodeJWTPayload(t, tok)
		assert.NotContains(t, claimScope(t, claims), "admin",
			"a scope outside the agent's ceiling must never be granted (non-widening)")
	})

	t.Run("deactivated_service_account_subject_cannot_seed_delegation", func(t *testing.T) {
		// The subject here is a DIFFERENT service account than the agent -
		// simulates a multi-hop chain where an upstream agent's own token is
		// being re-exchanged as the subject_token. Deactivate it AFTER
		// minting its token but before the exchange: the token is still
		// cryptographically valid (unexpired, correctly signed) - proving
		// this rejection actually comes from a live active-status check, not
		// from token validation catching something else.
		subjectClientID, subjectSecret := newDelegationAgent(t, ts, "openid,profile,email")
		subjectToken := agentAccessToken(t, ts, router, subjectClientID, subjectSecret)

		subjectClient, err := ts.StorageProvider.GetClientByClientID(context.Background(), subjectClientID)
		require.NoError(t, err)
		subjectClient.IsActive = false
		_, err = ts.StorageProvider.UpdateClient(context.Background(), subjectClient)
		require.NoError(t, err)

		clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
		actor := agentAccessToken(t, ts, router, clientID, secret)
		form := url.Values{}
		form.Set("grant_type", tokenExchangeGrant)
		form.Set("subject_token", subjectToken)
		form.Set("subject_token_type", accessTokenType)
		form.Set("actor_token", actor)
		form.Set("actor_token_type", accessTokenType)
		form.Set("resource", "https://api.example.com/v1")
		w := postTokenExchange(ts, router, form, clientID, secret)
		assert.Equal(t, http.StatusBadRequest, w.Code, "a deactivated service-account subject must not seed a delegation: %s", w.Body.String())
		assert.Contains(t, w.Body.String(), "no longer active")
	})

	t.Run("actor_token_must_belong_to_authenticated_agent", func(t *testing.T) {
		// A valid token that is NOT the agent's own (here a user token, whose
		// client_id != the agent) must be rejected as the actor (RFC 8693 §1.1).
		clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
		form := url.Values{}
		form.Set("grant_type", tokenExchangeGrant)
		form.Set("subject_token", testAccessToken(t, ts))
		form.Set("subject_token_type", accessTokenType)
		form.Set("actor_token", testAccessToken(t, ts)) // a user token, not the agent's
		form.Set("actor_token_type", accessTokenType)
		form.Set("resource", "https://api.example.com/v1")
		w := postTokenExchange(ts, router, form, clientID, secret)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "actor_token", "a foreign actor_token must be rejected")
	})
}
