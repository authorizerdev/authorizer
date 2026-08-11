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

	"github.com/authorizerdev/authorizer/internal/clientmetadata"
	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestCIMDConsentFlow covers ONE property: that a client_id which cannot be
// resolved is refused outright rather than falling through to the
// AllowedOrigins check, which would give an unresolvable client a laxer
// redirect_uri check than a resolved one.
//
// It is deliberately narrow. The end-to-end consent flow — page shown, decision
// honoured, code issued or access_denied returned — is covered by
// TestCIMDConsentEndToEnd in cimd_flow_test.go, which serves a real TLS document
// host. An earlier version of this comment claimed that coverage while the test
// only asserted the refusal, which is the kind of overstatement that makes a
// suite look stronger than it is.
func TestCIMDConsentFlow(t *testing.T) {
	cfg := getTestConfig()
	cfg.AuthorizerURL = "https://auth.example.com"
	cfg.EnableClientIDMetadataDocument = true
	ts := initTestSetup(t, cfg)

	// No document host is stood up: app.example.com does not resolve, so
	// resolution fails and the refusal below is what is under test. The resolved
	// path needs a real TLS host and lives in cimd_flow_test.go.
	const clientID = "https://app.example.com/client.json"
	const redirectURI = "http://localhost:3000/callback"

	router := gin.New()
	// Only the consent template: LoadHTMLGlob would also parse app.tmpl, which
	// uses the `json` FuncMap the real router registers and this one does not.
	router.LoadHTMLFiles("../../web/templates/consent.tmpl")
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/authorize/consent", ts.HttpProvider.ConsentHandler())

	authorizeQuery := func(state string) url.Values {
		qs := url.Values{}
		qs.Set("response_type", "code")
		qs.Set("client_id", clientID)
		qs.Set("redirect_uri", redirectURI)
		qs.Set("state", state)
		qs.Set("response_mode", "query")
		qs.Set("scope", "openid")
		qs.Set("code_challenge", s256Challenge("a-verifier-long-enough-to-be-valid-00000000000"))
		qs.Set("code_challenge_method", "S256")
		return qs
	}

	_, sessionToken := mcpSession(t, ts, []string{"openid"})

	t.Run("an unresolvable client_id is refused, not silently downgraded", func(t *testing.T) {
		// app.example.com does not serve a document, so resolution fails. The
		// request must be rejected rather than falling through to the
		// AllowedOrigins check — falling through would give an unresolvable
		// client a LAXER redirect_uri check than a resolved one.
		w := doAuthorizeGET(router, authorizeQuery("st"), sessionToken)
		require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())

		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "invalid_client", body["error"])
	})

	t.Run("IsMetadataClientID gates the whole path", func(t *testing.T) {
		// The discriminator between registry lookup and document fetch. An
		// ordinary client_id must never trigger an outbound request.
		assert.True(t, clientmetadata.IsMetadataClientID(clientID))
		assert.False(t, clientmetadata.IsMetadataClientID(cfg.ClientID))
	})
}

// TestConsentHandlerSecurityProperties pins the properties that make splitting
// one authorization across two requests safe. Each is asserted on its own
// because each closes a distinct hole.
func TestConsentHandlerSecurityProperties(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableClientIDMetadataDocument = true
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.LoadHTMLFiles("../../web/templates/consent.tmpl")
	router.POST("/authorize/consent", ts.HttpProvider.ConsentHandler())

	post := func(form url.Values, sessionToken string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/authorize/consent", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if sessionToken != "" {
			req.AddCookie(&http.Cookie{Name: constants.AppCookieName + "_session", Value: sessionToken})
		}
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("a missing consent_id is rejected", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, post(url.Values{"action": {"approve"}}, "").Code)
	})

	t.Run("an unknown consent_id is rejected", func(t *testing.T) {
		// Expired, already used and never-existed are deliberately
		// indistinguishable, so this is not an oracle for guessing ids.
		f := url.Values{"consent_id": {uuid.NewString()}, "action": {"approve"}}
		w := post(f, "")
		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "expired or was already used")
	})

	t.Run("a consent is single-use", func(t *testing.T) {
		// A replayed approval must not mint a second authorization code, so the
		// pending record is removed before the decision is acted on.
		id := uuid.NewString()
		payload := `{"client_id":"https://app.example.com/c.json","client_name":"n",` +
			`"redirect_uri":"http://localhost:3000/cb","query":"state=x","user_id":"someone"}`
		require.NoError(t, ts.MemoryStoreProvider.SetState("cimd_consent:"+id, payload))

		// First use: rejected on the session check (no cookie), but it must
		// still have consumed the record.
		first := post(url.Values{"consent_id": {id}, "action": {"approve"}}, "")
		assert.Equal(t, http.StatusUnauthorized, first.Code)

		second := post(url.Values{"consent_id": {id}, "action": {"approve"}}, "")
		require.Equal(t, http.StatusBadRequest, second.Code)
		assert.Contains(t, second.Body.String(), "expired or was already used",
			"the pending consent must be consumed on first use, whatever the outcome")
	})

	t.Run("a consent approved by a different session is refused", func(t *testing.T) {
		// Without this, a consent page rendered for one user could be submitted
		// in another user's browser and mint a code against their account.
		_, otherSession := mcpSession(t, ts, []string{"openid"})
		id := uuid.NewString()
		payload := `{"client_id":"https://app.example.com/c.json","client_name":"n",` +
			`"redirect_uri":"http://localhost:3000/cb","query":"state=x","user_id":"a-different-user"}`
		require.NoError(t, ts.MemoryStoreProvider.SetState("cimd_consent:"+id, payload))

		w := post(url.Values{"consent_id": {id}, "action": {"approve"}}, otherSession)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "access_denied")
	})
}
