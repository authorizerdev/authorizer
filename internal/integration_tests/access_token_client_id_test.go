package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// RFC 9068 §2.2: an access token names the client it was issued to in a
// client_id claim. Without it, once an RFC 8707 resource indicator replaces
// `aud`, nothing in the token identifies the client, so a resource server
// (e.g. an MCP server accepting tokens from several OAuth clients) cannot
// audit, rate-limit or apply policy per client.
//
// OIDC Core §2: an ID token's azp names the client it was issued to.
//
// Every subtest uses a client distinct from the deployment's bootstrap
// client_id, because that is also the value the claims fall back to: a test
// using the bootstrap client could not tell "the requesting client" from
// "the fallback".
func TestAccessTokenCarriesClientID(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	t.Run("authorization_code and refresh name the requesting client", func(t *testing.T) {
		clientID, secret := "cid-code-client", "cid-code-client-secret"
		registerTestClient(t, ts, clientID, secret)
		require.NotEqual(t, clientID, cfg.ClientID)
		router, code, verifier := loginForOfflineAccess(t, ts, clientID)

		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("code_verifier", verifier)
		form.Set("redirect_uri", "http://localhost:3000/callback")
		w := exchangeCode(router, form, []string{clientID, secret})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		body := decodeJSON(t, w)
		assertIssuedTo(t, body, clientID, clientID)

		refreshToken, _ := body["refresh_token"].(string)
		require.NotEmpty(t, refreshToken, "offline_access must yield a refresh token")
		refresh := url.Values{}
		refresh.Set("grant_type", "refresh_token")
		refresh.Set("refresh_token", refreshToken)
		w = exchangeCode(router, refresh, []string{clientID, secret})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
		assertIssuedTo(t, decodeJSON(t, w), clientID, clientID)
	})

	t.Run("a resource-bound token keeps the client beside the audience", func(t *testing.T) {
		clientID, secret := "cid-resource-client", "cid-resource-client-secret"
		registerTestClient(t, ts, clientID, secret)
		resource := "https://mcp.example.com/mcp"
		router, sessionToken := mcpSession(t, ts, []string{"openid"})
		verifier := "a-code-verifier-that-is-long-enough-to-be-valid-000000"

		qs := url.Values{}
		qs.Set("response_type", "code")
		qs.Set("client_id", clientID)
		qs.Set("redirect_uri", "http://localhost:3000/callback")
		qs.Set("state", "st")
		qs.Set("response_mode", "query")
		qs.Set("scope", "openid")
		qs.Set("code_challenge", s256Challenge(verifier))
		qs.Set("code_challenge_method", "S256")
		qs.Set("resource", resource)
		code := codeFromRedirect(t, doAuthorizeGET(router, qs, sessionToken))

		exchange := url.Values{}
		exchange.Set("grant_type", "authorization_code")
		exchange.Set("code", code)
		exchange.Set("redirect_uri", "http://localhost:3000/callback")
		exchange.Set("code_verifier", verifier)
		exchange.Set("resource", resource)
		w := exchangeCode(router, exchange, []string{clientID, secret})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		// The access token's aud is the resource; the ID token's stays the client.
		assertIssuedTo(t, decodeJSON(t, w), clientID, resource)
	})

	t.Run("client_credentials names the service account's public client_id", func(t *testing.T) {
		router := gin.New()
		router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

		// An explicit ClientID distinct from the row id, so the test fails if
		// the handler passed sa.ID (AddClient copies ID into an empty ClientID).
		secret := "cc-secret-" + uuid.NewString()
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), ccClientCost)
		require.NoError(t, err)
		sa, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
			ClientID:      "svc-cid-" + uuid.NewString(),
			Kind:          constants.ClientKindServiceAccount,
			Name:          "cid-sa-" + uuid.NewString(),
			ClientSecret:  string(hash),
			AllowedScopes: ccClientScope,
			IsActive:      true,
		})
		require.NoError(t, err)
		require.NotEqual(t, sa.ID, sa.ClientID)

		for _, resource := range []string{"", "https://api.example.com"} {
			form := url.Values{}
			form.Set("grant_type", constants.GrantTypeClientCredentials)
			if resource != "" {
				form.Set("resource", resource)
			}
			w := postClientCredentials(router, form, []string{sa.ClientID, secret})
			require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

			accessToken, _ := decodeJSON(t, w)["access_token"].(string)
			claims := decodeJWTPayloadUnverified(t, accessToken)
			assert.Equal(t, sa.ClientID, claims["client_id"], "resource=%q", resource)
			assert.Equal(t, sa.ID, claims["sub"], "resource=%q: sub is unchanged", resource)
			wantAud := resource
			if wantAud == "" {
				// Unchanged: without a resource a machine token's audience is
				// the deployment's client_id, not the service account.
				wantAud = cfg.ClientID
			}
			assert.Equal(t, wantAud, claims["aud"], "resource=%q", resource)
		}
	})

	t.Run("hosted-app login names the deployment client", func(t *testing.T) {
		_, ctx := createContext(ts)
		email := "cid_login_" + uuid.NewString() + "@authorizer.dev"
		password := "Password@123"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, res.AccessToken)
		require.NotNil(t, res.IDToken)

		access := decodeJWTPayloadUnverified(t, *res.AccessToken)
		assert.Equal(t, cfg.ClientID, access["client_id"])
		assert.Equal(t, cfg.ClientID, access["aud"], "aud is unchanged")
		id := decodeJWTPayloadUnverified(t, *res.IDToken)
		assert.Equal(t, cfg.ClientID, id["azp"])
		assert.Equal(t, cfg.ClientID, id["aud"], "aud is unchanged")
	})

	t.Run("discovery advertises azp", func(t *testing.T) {
		router := gin.New()
		router.GET("/.well-known/openid-configuration", ts.HttpProvider.OpenIDConfigurationHandler())
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
		req.Host = "localhost"
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var body struct {
			ClaimsSupported []string `json:"claims_supported"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Contains(t, body.ClaimsSupported, "azp")
		assert.Contains(t, body.ClaimsSupported, "aud", "existing entries are kept")
	})
}

// assertIssuedTo checks a token response: the access token's client_id and
// the ID token's azp both name clientID, and the access token's aud is
// accessAud.
func assertIssuedTo(t *testing.T, body map[string]interface{}, clientID, accessAud string) {
	t.Helper()
	accessToken, _ := body["access_token"].(string)
	require.NotEmpty(t, accessToken)
	access := decodeJWTPayloadUnverified(t, accessToken)
	assert.Equal(t, clientID, access["client_id"], "access token client_id")
	assert.Equal(t, accessAud, access["aud"], "access token aud")

	idToken, _ := body["id_token"].(string)
	require.NotEmpty(t, idToken)
	id := decodeJWTPayloadUnverified(t, idToken)
	assert.Equal(t, clientID, id["azp"], "id token azp")
	assert.Equal(t, clientID, id["aud"], "id token aud")
}

// TestClientIDAndAzpWithCustomAccessTokenScript pins backward compatibility
// for CustomAccessTokenScript: client_id was already reserved, so a script
// still cannot forge it; azp was not reserved, so a script that set it before
// these claims existed keeps its output in both tokens.
func TestClientIDAndAzpWithCustomAccessTokenScript(t *testing.T) {
	cfg := getTestConfig()
	cfg.CustomAccessTokenScript = `function(user, tokenPayload) {
		return { client_id: "forged", azp: "from-script", custom_claim: "kept" };
	}`
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "cid_script_" + uuid.NewString() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	require.NoError(t, err)
	require.NotNil(t, res.AccessToken)
	require.NotNil(t, res.IDToken)

	access := decodeJWTPayloadUnverified(t, *res.AccessToken)
	assert.Equal(t, cfg.ClientID, access["client_id"], "a script must not forge client_id")
	assert.Equal(t, "from-script", access["azp"], "a script-set azp is kept, as before")
	assert.Equal(t, "kept", access["custom_claim"])

	id := decodeJWTPayloadUnverified(t, *res.IDToken)
	assert.Equal(t, "from-script", id["azp"], "a script-set azp is kept, as before")
}
