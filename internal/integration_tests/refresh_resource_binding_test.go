package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// s256Challenge is the RFC 7636 S256 code challenge for a verifier.
func s256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// postToken drives POST /oauth/token with form-encoded params and returns the
// decoded body.
func postToken(t *testing.T, router http.Handler, form url.Values) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

// TestRefreshPreservesTheResourceBinding pins RFC 8707 §2.2 across rotation.
//
// The resource indicator a client sends to /authorize becomes the access token's
// `aud`, which is what stops a token minted for one resource server being
// replayed at another. That binding used to survive exactly one token: the local
// `resource` in the token endpoint was populated only inside the
// authorization_code branch, so on refresh it was empty, accessTokenAudience fell
// back to the client id, and the rotated token came back UNBOUND — usable at
// Authorizer's own API, which is precisely what the restriction existed to
// prevent.
//
// The failure mode hid itself. The first token is correct, so every manual test
// and every demo passes; only the refreshed one is wrong, and only after the
// access token's lifetime has elapsed. For the MCP surface — whose tokens are
// audience-bound by specification and whose clients refresh proactively before
// expiry — it meant a connection that worked and then failed permanently.
func TestRefreshPreservesTheResourceBinding(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	resource := "https://mcp.example.com/mcp"
	router, sessionToken := mcpSession(t, ts, []string{"openid", "offline_access"})

	verifier := "a-code-verifier-that-is-long-enough-to-be-valid-000000"
	challenge := s256Challenge(verifier)

	qs := url.Values{}
	qs.Set("response_type", "code")
	qs.Set("client_id", cfg.ClientID)
	qs.Set("redirect_uri", "http://localhost:3000/callback")
	qs.Set("state", "st")
	qs.Set("response_mode", "query")
	qs.Set("scope", "openid offline_access")
	qs.Set("code_challenge", challenge)
	qs.Set("code_challenge_method", "S256")
	qs.Set("resource", resource)

	code := codeFromRedirect(t, doAuthorizeGET(router, qs, sessionToken))

	exchange := url.Values{}
	exchange.Set("grant_type", "authorization_code")
	exchange.Set("code", code)
	exchange.Set("client_id", cfg.ClientID)
	exchange.Set("redirect_uri", "http://localhost:3000/callback")
	exchange.Set("code_verifier", verifier)
	exchange.Set("resource", resource)

	status, body := postToken(t, router, exchange)
	require.Equal(t, http.StatusOK, status, "body: %v", body)

	firstAccess, _ := body["access_token"].(string)
	refreshToken, _ := body["refresh_token"].(string)
	require.NotEmpty(t, firstAccess)
	require.NotEmpty(t, refreshToken, "offline_access must yield a refresh token, or this test proves nothing")

	firstClaims, err := ts.TokenProvider.ParseJWTToken(firstAccess)
	require.NoError(t, err)
	require.Equal(t, resource, firstClaims["aud"],
		"the authorization_code path must bind the audience — the regression under test is about what happens NEXT")

	t.Run("a refresh naming a different resource is refused", func(t *testing.T) {
		// Runs FIRST: rejection happens before rotation, so the success case
		// below still has a live refresh token. Refresh tokens rotate on use.
		//
		// RFC 8707 §2.2 lets a refresh restrict the resource, never switch it.
		// Silently ignoring the mismatch would hand back a token for a resource
		// the caller did not ask for.
		refresh := url.Values{}
		refresh.Set("grant_type", "refresh_token")
		refresh.Set("refresh_token", refreshToken)
		refresh.Set("client_id", cfg.ClientID)
		refresh.Set("resource", "https://attacker.example.com/mcp")

		status, body := postToken(t, router, refresh)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "invalid_target", body["error"])
	})
	t.Run("the rotated access token keeps the audience", func(t *testing.T) {
		refresh := url.Values{}
		refresh.Set("grant_type", "refresh_token")
		refresh.Set("refresh_token", refreshToken)
		refresh.Set("client_id", cfg.ClientID)

		status, body := postToken(t, router, refresh)
		require.Equal(t, http.StatusOK, status, "body: %v", body)

		rotated, _ := body["access_token"].(string)
		require.NotEmpty(t, rotated)

		claims, pErr := ts.TokenProvider.ParseJWTToken(rotated)
		require.NoError(t, pErr)
		assert.Equal(t, resource, claims["aud"],
			"a refreshed token must stay bound to the resource the grant named; falling back to the client id "+
				"silently widens a token the user scoped to one resource server")
	})

}

// TestRefreshWithoutResourceIsUnchanged is the regression guard for every
// deployment that has never used a resource indicator: their refresh tokens
// carry no `resource` claim, and the rotated access token must keep the client
// id as its audience exactly as before.
func TestRefreshWithoutResourceIsUnchanged(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router, sessionToken := mcpSession(t, ts, []string{"openid", "offline_access"})
	verifier := "another-code-verifier-long-enough-to-be-valid-00000000"

	qs := url.Values{}
	qs.Set("response_type", "code")
	qs.Set("client_id", cfg.ClientID)
	qs.Set("redirect_uri", "http://localhost:3000/callback")
	qs.Set("state", "st")
	qs.Set("response_mode", "query")
	qs.Set("scope", "openid offline_access")
	qs.Set("code_challenge", s256Challenge(verifier))
	qs.Set("code_challenge_method", "S256")

	code := codeFromRedirect(t, doAuthorizeGET(router, qs, sessionToken))

	exchange := url.Values{}
	exchange.Set("grant_type", "authorization_code")
	exchange.Set("code", code)
	exchange.Set("client_id", cfg.ClientID)
	exchange.Set("redirect_uri", "http://localhost:3000/callback")
	exchange.Set("code_verifier", verifier)

	status, body := postToken(t, router, exchange)
	require.Equal(t, http.StatusOK, status, "body: %v", body)
	refreshToken, _ := body["refresh_token"].(string)
	require.NotEmpty(t, refreshToken)

	refresh := url.Values{}
	refresh.Set("grant_type", "refresh_token")
	refresh.Set("refresh_token", refreshToken)
	refresh.Set("client_id", cfg.ClientID)

	status, body = postToken(t, router, refresh)
	require.Equal(t, http.StatusOK, status, "body: %v", body)

	rotated, _ := body["access_token"].(string)
	claims, err := ts.TokenProvider.ParseJWTToken(rotated)
	require.NoError(t, err)
	assert.Equal(t, cfg.ClientID, claims["aud"],
		"an unbound grant must keep the client id as the audience — the fix must not stamp an empty resource")
}
