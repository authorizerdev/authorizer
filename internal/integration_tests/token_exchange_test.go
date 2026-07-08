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

func TestTokenExchangeDelegation(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	baseForm := func(agentScopeCeiling string) (url.Values, string, string) {
		clientID, secret := newDelegationAgent(t, ts, agentScopeCeiling)
		subject := testAccessToken(t, ts)
		actor := testAccessToken(t, ts)
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
		scope, _ := claims["scope"].(string)
		assert.NotContains(t, strings.Fields(scope), "admin",
			"a scope outside the agent's ceiling must never be granted (non-widening)")
	})
}
