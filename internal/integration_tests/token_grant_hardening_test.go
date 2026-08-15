package integration_tests

import (
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

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
)

// TestClientCredentialsAudienceBinding covers RFC 8707 audience binding on the
// client_credentials grant.
//
// Without it every service-account token in a deployment carries the same `aud`
// (the deployment's global client_id), so a token minted for one internal
// service is equally valid at every other — the resource server loses the
// cheapest rejection it has. Binding a machine token to the API it is meant for
// is the established shape for this grant; `resource` (RFC 8707) is the
// standards-track spelling of it, and some providers expose the same idea as an
// `audience` request parameter or an audience claim mapper.
func TestClientCredentialsAudienceBinding(t *testing.T) {
	ts := initTestSetup(t, getTestConfig())
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	saID, saSecret := newDelegationAgent(t, ts, "read:orders")

	get := func(f url.Values) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(f.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
		req.SetBasicAuth(saID, saSecret)
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("resource binds the token audience", func(t *testing.T) {
		f := url.Values{}
		f.Set("grant_type", "client_credentials")
		f.Set("resource", "https://orders.internal")
		w := get(f)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var res map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		claims := decodeJWTPayload(t, res["access_token"].(string))
		assert.Equal(t, "https://orders.internal", claims["aud"],
			"the requested resource must become the token audience")
	})

	t.Run("omitting resource preserves the previous global audience", func(t *testing.T) {
		f := url.Values{}
		f.Set("grant_type", "client_credentials")
		w := get(f)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var res map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		claims := decodeJWTPayload(t, res["access_token"].(string))
		assert.Equal(t, ts.Config.ClientID, claims["aud"],
			"backward compatibility: no resource means the deployment client_id stays the audience")
	})

	t.Run("rejects a non-absolute-URI resource", func(t *testing.T) {
		for _, bad := range []string{"not-a-uri", "https://rs.example.com#frag", "../../etc"} {
			f := url.Values{}
			f.Set("grant_type", "client_credentials")
			f.Set("resource", bad)
			w := get(f)
			assert.Equal(t, http.StatusBadRequest, w.Code, "resource %q must be rejected", bad)
			assert.Contains(t, w.Body.String(), "invalid_target")
		}
	})

	t.Run("rejects a repeated resource parameter", func(t *testing.T) {
		body := "grant_type=client_credentials&resource=https%3A%2F%2Fa.example.com&resource=https%3A%2F%2Fb.example.com"
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
		req.SetBasicAuth(saID, saSecret)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code,
			"a multi-audience machine token would be replayable across resource servers")
	})
}

// TestTokenExchangeRejectsInvalidResourceIndicator pins RFC 8707 §2 on the
// token-exchange grant. /authorize enforced this from the start; the exchange
// path did not, so an agent could bind a delegated token's `aud` to an arbitrary
// opaque string and make the audience restriction unenforceable downstream.
func TestTokenExchangeRejectsInvalidResourceIndicator(t *testing.T) {
	ts := initTestSetup(t, getTestConfig())
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	_, ctx := createContext(ts)
	email := "res_ind_" + uuid.New().String() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: "Password@123", ConfirmPassword: "Password@123",
	})
	require.NoError(t, err)
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	subjectTok, err := ts.TokenProvider.CreateAuthToken(nil, &token.AuthTokenConfig{
		User: user, Roles: []string{"user"}, Scope: []string{"openid"},
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		Nonce:       uuid.New().String(), HostName: testAuthorizerHost(ts),
	})
	require.NoError(t, err)
	// Register the session CreateAuthToken does not write. Every production path
	// that mints an access token registers it (login, signup, verify_email,
	// authorization_code, refresh, client_credentials), so a token without an
	// entry is a token whose session has been revoked — and token exchange now
	// refuses to seed a delegation from one. Without this the subject_token here
	// is revoked-shaped, and this test would fail on the happy-path case it exists
	// to protect for a reason that has nothing to do with resource indicators.
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		constants.AuthRecipeMethodBasicAuth+":"+user.ID,
		constants.TokenTypeAccessToken+"_"+subjectTok.FingerPrint,
		subjectTok.AccessToken.Token,
		subjectTok.AccessToken.ExpiresAt,
	))

	agentID, agentSecret := newDelegationAgent(t, ts, "openid")
	actorTok := agentAccessToken(t, ts, router, agentID, agentSecret)

	mk := func(resource string) url.Values {
		f := url.Values{}
		f.Set("grant_type", tokenExchangeGrant)
		f.Set("subject_token", subjectTok.AccessToken.Token)
		f.Set("subject_token_type", accessTokenType)
		f.Set("actor_token", actorTok)
		f.Set("actor_token_type", accessTokenType)
		f.Set("resource", resource)
		return f
	}

	for _, bad := range []string{"not-a-uri", "https://rs.example.com#frag", "../../etc"} {
		w := postTokenExchange(ts, router, mk(bad), agentID, agentSecret)
		assert.Equal(t, http.StatusBadRequest, w.Code, "resource %q must be rejected", bad)
		assert.Contains(t, w.Body.String(), "invalid_target")
	}

	// A valid absolute URI still works — the check must not break the happy path.
	w := postTokenExchange(ts, router, mk("https://mcp.example.com/v1"), agentID, agentSecret)
	assert.Equal(t, http.StatusOK, w.Code, "a valid resource indicator must still be accepted: %s", w.Body.String())
}
