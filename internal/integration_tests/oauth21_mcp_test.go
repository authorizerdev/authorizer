package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
)

// mcpSession signs up a fresh user, registers a browser session, and returns a
// router with /authorize + /oauth/token mounted plus the session cookie value.
// Tests build their own /authorize query (response_type, resource, PKCE method)
// on top of it.
func mcpSession(t *testing.T, ts *testSetup, scope []string) (http.Handler, string) {
	t.Helper()
	_, ctx := createContext(ts)

	email := "mcp_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	nonce := uuid.New().String()
	sessionData, sessionToken, sessionExpiresAt, err := ts.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       ts.Config.DefaultRoles,
		Scope:       scope,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionKey, constants.TokenTypeSessionToken+"_"+sessionData.Nonce, sessionToken, sessionExpiresAt))

	router := gin.New()
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	return router, sessionToken
}

func doAuthorizeGET(router http.Handler, qs url.Values, sessionToken string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/authorize?"+qs.Encode(), nil)
	req.AddCookie(&http.Cookie{Name: constants.AppCookieName + "_session", Value: sessionToken})
	router.ServeHTTP(w, req)
	return w
}

func codeFromRedirect(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	require.Equal(t, http.StatusFound, w.Code, "authorize should redirect: %s", w.Body.String())
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code, "authorization code must be present")
	return code
}

// Item 2: RFC 8707 resource indicator on the primary authorization_code flow.
// A resource bound at /authorize must become the access token's `aud`, while
// the id_token audience stays the requesting client.
func TestOAuth21_ResourceIndicator_BindsAccessTokenAudience(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "mcp-client", "mcp-secret")

	const resource = "https://mcp.example.com"
	scope := []string{"openid", "profile", "email"}

	verifier := uuid.New().String() + uuid.New().String()
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	router, cookie := mcpSession(t, ts, scope)

	qs := url.Values{}
	qs.Set("client_id", "mcp-client")
	qs.Set("response_type", "code")
	qs.Set("redirect_uri", "http://localhost:3000/callback")
	qs.Set("code_challenge", challenge)
	qs.Set("code_challenge_method", "S256")
	qs.Set("state", uuid.New().String())
	qs.Set("response_mode", "query")
	qs.Set("scope", "openid profile email")
	qs.Set("resource", resource)

	code := codeFromRedirect(t, doAuthorizeGET(router, qs, cookie))

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", "http://localhost:3000/callback")
	form.Set("resource", resource)

	w := exchangeCode(router, form, []string{"mcp-client", "mcp-secret"})
	require.Equal(t, http.StatusOK, w.Code, "code exchange body: %s", w.Body.String())

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	accessToken, _ := body["access_token"].(string)
	require.NotEmpty(t, accessToken)
	assert.Equal(t, resource, decodeJWTPayloadUnverified(t, accessToken)["aud"],
		"access_token aud must be the bound resource (RFC 8707)")

	// The id_token audience must remain the OAuth client (OIDC), not the resource.
	idToken, _ := body["id_token"].(string)
	require.NotEmpty(t, idToken)
	assert.Equal(t, "mcp-client", decodeJWTPayloadUnverified(t, idToken)["aud"],
		"id_token aud must stay the requesting client, not the resource")
}

// Item 2: a resource bound at /authorize must be echoed and matched at
// /oauth/token — a mismatch (or an omitted resource) is rejected invalid_grant,
// exactly like a PKCE code_verifier mismatch.
func TestOAuth21_ResourceIndicator_MismatchRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "mcp-client", "mcp-secret")

	scope := []string{"openid", "profile", "email"}

	authorizeForCode := func() (http.Handler, string, string) {
		verifier := uuid.New().String() + uuid.New().String()
		sum := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(sum[:])
		router, cookie := mcpSession(t, ts, scope)
		qs := url.Values{}
		qs.Set("client_id", "mcp-client")
		qs.Set("response_type", "code")
		qs.Set("redirect_uri", "http://localhost:3000/callback")
		qs.Set("code_challenge", challenge)
		qs.Set("code_challenge_method", "S256")
		qs.Set("state", uuid.New().String())
		qs.Set("response_mode", "query")
		qs.Set("scope", "openid profile email")
		qs.Set("resource", "https://mcp.example.com")
		return router, codeFromRedirect(t, doAuthorizeGET(router, qs, cookie)), verifier
	}

	baseForm := func(code, verifier string) url.Values {
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("code_verifier", verifier)
		form.Set("redirect_uri", "http://localhost:3000/callback")
		return form
	}

	t.Run("wrong resource rejected", func(t *testing.T) {
		router, code, verifier := authorizeForCode()
		form := baseForm(code, verifier)
		form.Set("resource", "https://evil.example.com")
		w := exchangeCode(router, form, []string{"mcp-client", "mcp-secret"})
		assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		var errBody map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errBody))
		assert.Equal(t, "invalid_grant", errBody["error"])
	})

	t.Run("omitted resource rejected", func(t *testing.T) {
		router, code, verifier := authorizeForCode()
		form := baseForm(code, verifier) // no resource echoed
		w := exchangeCode(router, form, []string{"mcp-client", "mcp-secret"})
		assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		var errBody map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errBody))
		assert.Equal(t, "invalid_grant", errBody["error"])
	})
}

// Item 1: OAuth 2.1 refresh-token rotation with reuse invalidation. After a
// refresh rotates the token, replaying the OLD refresh token must be rejected
// invalid_grant AND trigger the breach response — the whole live session
// family is revoked, so the freshly-issued refresh token stops working too.
func TestOAuth21_RefreshTokenReuse_RevokesFamily(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "reuse-client", "reuse-secret")

	router, code, verifier := loginForOfflineAccess(t, ts, "reuse-client")

	exchangeForm := url.Values{}
	exchangeForm.Set("grant_type", "authorization_code")
	exchangeForm.Set("code", code)
	exchangeForm.Set("code_verifier", verifier)
	exchangeForm.Set("redirect_uri", "http://localhost:3000/callback")

	w := exchangeCode(router, exchangeForm, []string{"reuse-client", "reuse-secret"})
	require.Equal(t, http.StatusOK, w.Code, "code exchange body: %s", w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	oldRefresh, _ := body["refresh_token"].(string)
	require.NotEmpty(t, oldRefresh, "offline_access scope must yield a refresh_token")

	// Rotate once — a new refresh token is issued and the old one is retired.
	refresh := func(rt string) *httptest.ResponseRecorder {
		f := url.Values{}
		f.Set("grant_type", "refresh_token")
		f.Set("refresh_token", rt)
		return exchangeCode(router, f, []string{"reuse-client", "reuse-secret"})
	}

	w1 := refresh(oldRefresh)
	require.Equal(t, http.StatusOK, w1.Code, "first refresh body: %s", w1.Body.String())
	var body1 map[string]interface{}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &body1))
	newRefresh, _ := body1["refresh_token"].(string)
	require.NotEmpty(t, newRefresh)
	require.NotEqual(t, oldRefresh, newRefresh, "rotation must mint a fresh refresh token")

	// Replay the OLD (already-rotated) refresh token: reuse detected.
	w2 := refresh(oldRefresh)
	assert.Equal(t, http.StatusBadRequest, w2.Code, "reuse body: %s", w2.Body.String())
	var errBody map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &errBody))
	assert.Equal(t, "invalid_grant", errBody["error"])

	// Breach response: the reuse revoked the whole family, so the token that
	// was legitimately rotated in is now dead too.
	w3 := refresh(newRefresh)
	assert.Equal(t, http.StatusBadRequest, w3.Code,
		"the live refresh token must be revoked after reuse of its rotated predecessor: %s", w3.Body.String())
	var errBody3 map[string]interface{}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &errBody3))
	assert.Equal(t, "invalid_grant", errBody3["error"])
}

// Item 4: the --oauth21-strict flag. Default (false) leaves the implicit grant
// and PKCE "plain" working exactly as today; true rejects both.
func TestOAuth21_StrictMode_Gating(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "strict-client", "strict-secret")

	implicitQS := func() url.Values {
		qs := url.Values{}
		qs.Set("client_id", "strict-client")
		qs.Set("response_type", "token")
		qs.Set("redirect_uri", "http://localhost:3000/callback")
		qs.Set("state", uuid.New().String())
		qs.Set("response_mode", "fragment")
		qs.Set("scope", "openid profile email")
		return qs
	}

	plainPKCEQS := func(verifier string) url.Values {
		qs := url.Values{}
		qs.Set("client_id", "strict-client")
		qs.Set("response_type", "code")
		qs.Set("redirect_uri", "http://localhost:3000/callback")
		// plain: code_challenge == code_verifier (RFC 7636 §4.2).
		qs.Set("code_challenge", verifier)
		qs.Set("code_challenge_method", "plain")
		qs.Set("state", uuid.New().String())
		qs.Set("response_mode", "query")
		qs.Set("scope", "openid profile email")
		return qs
	}

	t.Run("default false: implicit response_type=token still works", func(t *testing.T) {
		ts.Config.OAuth21Strict = false
		router, cookie := mcpSession(t, ts, []string{"openid", "profile", "email"})
		w := doAuthorizeGET(router, implicitQS(), cookie)
		require.Equal(t, http.StatusFound, w.Code, "body: %s", w.Body.String())
		assert.Contains(t, w.Header().Get("Location"), "access_token=",
			"implicit flow must return an access token in the fragment when strict is off")
	})

	t.Run("strict true: implicit response_type=token rejected", func(t *testing.T) {
		ts.Config.OAuth21Strict = true
		router, cookie := mcpSession(t, ts, []string{"openid", "profile", "email"})
		w := doAuthorizeGET(router, implicitQS(), cookie)
		require.Equal(t, http.StatusFound, w.Code, "body: %s", w.Body.String())
		loc := w.Header().Get("Location")
		assert.Contains(t, loc, "unsupported_response_type")
		assert.NotContains(t, loc, "access_token=")
	})

	t.Run("default false: PKCE plain still works", func(t *testing.T) {
		ts.Config.OAuth21Strict = false
		verifier := uuid.New().String() + uuid.New().String()
		router, cookie := mcpSession(t, ts, []string{"openid", "profile", "email"})
		code := codeFromRedirect(t, doAuthorizeGET(router, plainPKCEQS(verifier), cookie))

		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("code_verifier", verifier)
		form.Set("redirect_uri", "http://localhost:3000/callback")
		w := exchangeCode(router, form, []string{"strict-client", "strict-secret"})
		require.Equal(t, http.StatusOK, w.Code, "plain PKCE exchange body: %s", w.Body.String())
	})

	t.Run("strict true: PKCE plain rejected at authorize", func(t *testing.T) {
		ts.Config.OAuth21Strict = true
		verifier := uuid.New().String() + uuid.New().String()
		router, cookie := mcpSession(t, ts, []string{"openid", "profile", "email"})
		w := doAuthorizeGET(router, plainPKCEQS(verifier), cookie)
		assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		assert.Contains(t, w.Body.String(), "invalid_request")
		assert.Contains(t, w.Body.String(), "plain")
	})

	// Reset so a shared cfg pointer can't leak strict mode into later tests.
	ts.Config.OAuth21Strict = false
}

// Item 3 / Item 4: the RFC 8414 metadata handler advertises resource indicators
// and reflects strict-mode PKCE. The route alias itself is a one-line wrapper of
// this same handler (see internal/server/http_routes.go).
func TestOAuth21_AuthServerMetadata(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/.well-known/oauth-authorization-server", ts.HttpProvider.OpenIDConfigurationHandler())

	fetch := func() map[string]interface{} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		var doc map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))
		return doc
	}

	methodsOf := func(doc map[string]interface{}) []string {
		raw, _ := doc["code_challenge_methods_supported"].([]interface{})
		out := make([]string, 0, len(raw))
		for _, m := range raw {
			out = append(out, m.(string))
		}
		return out
	}

	ts.Config.OAuth21Strict = false
	doc := fetch()
	assert.Equal(t, true, doc["resource_indicators_supported"],
		"metadata must advertise resource_indicators_supported")
	assert.ElementsMatch(t, []string{"S256", "plain"}, methodsOf(doc))

	ts.Config.OAuth21Strict = true
	assert.ElementsMatch(t, []string{"S256"}, methodsOf(fetch()),
		"strict mode must advertise S256 only")

	ts.Config.OAuth21Strict = false
}
