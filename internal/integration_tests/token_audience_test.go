package integration_tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeJWTPayloadUnverified extracts the claims of a JWT without checking its
// signature — sufficient for tests that only need to read a claim from a
// token this same test just received from a trusted local server.
func decodeJWTPayloadUnverified(t *testing.T, tok string) map[string]interface{} {
	t.Helper()
	parts := strings.Split(tok, ".")
	require.Len(t, parts, 3, "not a JWT: %s", tok)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &claims))
	return claims
}

// OIDC Core §2 / RFC 7519 §4.1.3: the "aud" claim MUST identify the intended
// recipient — the OAuth client the token was issued to. Before this fix,
// id_token/access_token/refresh_token minted directly by /authorize (hybrid,
// implicit) and by /oauth/token's authorization_code/refresh_token grants
// were audienced to the single reserved bootstrap client_id regardless of
// which client actually requested them, so a spec-compliant RP other than
// the reserved client would reject its own id_token as not meant for it.
// Regression test for the oidcc-refresh-token conformance failure
// (ValidateIdToken: "'aud' is not our client id") surfaced once a second,
// non-reserved client was exercised end-to-end.
func TestTokenAudience_MatchesRequestingClient_NotReservedClient(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	registerTestClient(t, ts, "aud-test-client", "aud-test-client-secret")
	require.NotEqual(t, "aud-test-client", cfg.ClientID, "test client must be distinct from the reserved bootstrap client")

	router, code, codeVerifier := loginForOfflineAccess(t, ts, "aud-test-client")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("redirect_uri", "http://localhost:3000/callback")

	w := exchangeCode(router, form, []string{"aud-test-client", "aud-test-client-secret"})
	require.Equal(t, http.StatusOK, w.Code, "code exchange body: %s", w.Body.String())

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	idToken, _ := body["id_token"].(string)
	require.NotEmpty(t, idToken)
	assert.Equal(t, "aud-test-client", decodeJWTPayloadUnverified(t, idToken)["aud"],
		"id_token aud must be the requesting client, not the reserved bootstrap client")

	accessToken, _ := body["access_token"].(string)
	require.NotEmpty(t, accessToken)
	assert.Equal(t, "aud-test-client", decodeJWTPayloadUnverified(t, accessToken)["aud"],
		"access_token aud must be the requesting client, not the reserved bootstrap client")

	refreshToken, _ := body["refresh_token"].(string)
	require.NotEmpty(t, refreshToken)
	assert.Equal(t, "aud-test-client", decodeJWTPayloadUnverified(t, refreshToken)["aud"],
		"refresh_token aud must be the requesting client, not the reserved bootstrap client")
}
