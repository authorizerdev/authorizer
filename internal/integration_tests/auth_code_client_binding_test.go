package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Audit finding #7: the authorization code was not bound to the client it was
// issued to (RFC 6749 §4.1.3 — "ensure that the authorization code was issued to
// the authenticated confidential client").
//
// The stored code state carried the PKCE challenge, session, nonce, redirect_uri
// and resource, but no client identity, and the token endpoint never checked
// one. A code was therefore bound to a redirect_uri but not to an identity, so
// two confidential clients sharing a redirect origin could redeem each other's
// codes — the mix-up shape the clause exists to prevent.
func TestAuthCode_CrossClientRedemption_Rejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	registerTestClient(t, ts, "code-client-one", "code-client-one-secret")
	registerTestClient(t, ts, "code-client-two", "code-client-two-secret")

	router, code, codeVerifier := loginForOfflineAccess(t, ts, "code-client-one")

	t.Run("a different client redeeming the code is rejected", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("code_verifier", codeVerifier)
		form.Set("redirect_uri", "http://localhost:3000/callback")

		// client-two authenticates correctly as itself — the only thing wrong is
		// that this code belongs to client-one.
		w := exchangeCode(router, form, []string{"code-client-two", "code-client-two-secret"})
		assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		var errBody map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errBody))
		assert.Equal(t, "invalid_grant", errBody["error"])
		assert.Contains(t, errBody["error_description"], "not issued to this client")
	})

	t.Run("the issuing client can still redeem its own code", func(t *testing.T) {
		// The control: the guard must reject the wrong client without breaking
		// the right one. Uses a freshly minted code because the attempt above
		// consumed nothing — codes are single-use via GetAndRemoveState, so the
		// rejected exchange already burned that one.
		router, freshCode, freshVerifier := loginForOfflineAccess(t, ts, "code-client-one")

		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", freshCode)
		form.Set("code_verifier", freshVerifier)
		form.Set("redirect_uri", "http://localhost:3000/callback")

		w := exchangeCode(router, form, []string{"code-client-one", "code-client-one-secret"})
		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})
}
