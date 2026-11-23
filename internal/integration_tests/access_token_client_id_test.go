package integration_tests

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// RFC 9068 §2.2: an access token names the client it was issued to in a
// client_id claim. Without it, once an RFC 8707 resource indicator replaces
// `aud`, nothing in the token identifies the client, so a resource server
// (e.g. an MCP server accepting tokens from several OAuth clients) cannot
// audit, rate-limit or apply policy per client.
func TestAccessTokenCarriesClientID(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	t.Run("authorization_code names the requesting client", func(t *testing.T) {
		registerTestClient(t, ts, "cid-test-client", "cid-test-client-secret")
		router, code, verifier := loginForOfflineAccess(t, ts, "cid-test-client")

		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("code_verifier", verifier)
		form.Set("redirect_uri", "http://localhost:3000/callback")
		w := exchangeCode(router, form, []string{"cid-test-client", "cid-test-client-secret"})
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		body := decodeJSON(t, w)
		accessToken, _ := body["access_token"].(string)
		require.NotEmpty(t, accessToken)
		assert.Equal(t, "cid-test-client", decodeJWTPayloadUnverified(t, accessToken)["client_id"])
	})

	t.Run("authorization_code with a resource keeps the client beside the audience", func(t *testing.T) {
		resource := "https://mcp.example.com/mcp"
		router, sessionToken := mcpSession(t, ts, []string{"openid"})
		verifier := "a-code-verifier-that-is-long-enough-to-be-valid-000000"

		qs := url.Values{}
		qs.Set("response_type", "code")
		qs.Set("client_id", cfg.ClientID)
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
		exchange.Set("client_id", cfg.ClientID)
		exchange.Set("redirect_uri", "http://localhost:3000/callback")
		exchange.Set("code_verifier", verifier)
		exchange.Set("resource", resource)
		status, body := postToken(t, router, exchange)
		require.Equal(t, http.StatusOK, status, "body: %v", body)

		accessToken, _ := body["access_token"].(string)
		claims := decodeJWTPayloadUnverified(t, accessToken)
		require.Equal(t, resource, claims["aud"], "precondition: aud is the resource, not the client")
		assert.Equal(t, cfg.ClientID, claims["client_id"])
	})

	t.Run("client_credentials names the service account and keeps its audience", func(t *testing.T) {
		router := gin.New()
		router.POST("/oauth/token", ts.HttpProvider.TokenHandler())
		id, secret := createTestClient(t, ts, ccClientScope, true)

		for _, resource := range []string{"", "https://api.example.com"} {
			form := url.Values{}
			form.Set("grant_type", constants.GrantTypeClientCredentials)
			if resource != "" {
				form.Set("resource", resource)
			}
			w := postClientCredentials(router, form, []string{id, secret})
			require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

			accessToken, _ := decodeJSON(t, w)["access_token"].(string)
			claims := decodeJWTPayloadUnverified(t, accessToken)
			assert.Equal(t, id, claims["client_id"], "resource=%q", resource)
			wantAud := resource
			if wantAud == "" {
				// Unchanged: without a resource a machine token's audience is
				// the deployment's client_id, not the service account.
				wantAud = cfg.ClientID
			}
			assert.Equal(t, wantAud, claims["aud"], "resource=%q", resource)
		}
	})
}
