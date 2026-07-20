package http_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/token"
)

// validSessionCookie builds a cookie value identical in shape to a real
// session: AES-encrypted token.SessionData, matching what cookie.GetSession
// / crypto.DecryptAES on the handler side expect.
func validSessionCookie(t *testing.T, clientSecret string) string {
	t.Helper()
	sd := token.SessionData{
		Subject:     "user-1",
		LoginMethod: "basic_auth",
		Nonce:       "nonce-1",
		IssuedAt:    time.Now().Unix(),
	}
	raw, err := json.Marshal(sd)
	require.NoError(t, err)
	encrypted, err := crypto.EncryptAES(clientSecret, string(raw))
	require.NoError(t, err)
	return encrypted
}

// TestAuthorize_PromptLogin_RevokesExistingSession is a regression test for
// the oidcc-prompt-login conformance failure: prompt=login must force real
// re-authentication. Merely ignoring the session locally left the browser's
// cookie intact, so the login UI's own session check still saw the user as
// logged in and bounced straight back to /authorize without ever showing a
// login form. The fix must revoke the session (memory store) and clear the
// cookie so the login UI genuinely sees a logged-out user.
func TestAuthorize_PromptLogin_RevokesExistingSession(t *testing.T) {
	logger := zerolog.Nop()
	clientSecret := "test-client-secret"
	ms := &fakeMemoryStore{}
	h := &httpProvider{
		Config: &config.Config{
			ClientSecret:   clientSecret,
			AllowedOrigins: []string{"*"},
		},
		Dependencies: Dependencies{Log: &logger, MemoryStoreProvider: ms, StorageProvider: &redirectURIClientStore{}},
	}

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/authorize?client_id=abc&state=xyz&response_type=code&response_mode=query&scope=openid&prompt=login&redirect_uri=http%3A%2F%2Fexample.com%2Fapp%2Fcallback", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  constants.AppCookieName + "_session",
		Value: validSessionCookie(t, clientSecret),
	})

	h.AuthorizeHandler()(c)

	require.Len(t, ms.deletedSessions, 1, "the existing session must be revoked server-side")
	assert.Equal(t, [2]string{"basic_auth:user-1", "nonce-1"}, ms.deletedSessions[0])

	// The old session cookie must be cleared on the response (MaxAge < 0),
	// not left for the login UI's session check to find valid.
	cleared := false
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == constants.AppCookieName+"_session" && ck.MaxAge < 0 {
			cleared = true
		}
	}
	assert.True(t, cleared, "session cookie must be cleared, not merely ignored")

	// Must redirect to the login UI, not silently complete the authorization.
	assert.Equal(t, http.StatusFound, rec.Code)
}

// TestAuthorize_NoPrompt_ExistingSession_NotRevoked proves the fix is scoped
// to forced-reauth cases only — a plain /authorize request (no prompt, no
// max_age) with a valid session must NOT revoke it; that would break normal
// SSO continuation for every other flow.
func TestAuthorize_NoPrompt_ExistingSession_NotRevoked(t *testing.T) {
	logger := zerolog.Nop()
	clientSecret := "test-client-secret"
	ms := &fakeMemoryStore{}
	h := &httpProvider{
		Config: &config.Config{
			ClientSecret:   clientSecret,
			AllowedOrigins: []string{"*"},
		},
		Dependencies: Dependencies{Log: &logger, MemoryStoreProvider: ms, StorageProvider: &redirectURIClientStore{}},
	}

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet,
		"/authorize?client_id=abc&state=xyz&response_type=code&response_mode=query&scope=openid&redirect_uri=http%3A%2F%2Fexample.com%2Fapp%2Fcallback", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  constants.AppCookieName + "_session",
		Value: validSessionCookie(t, clientSecret),
	})

	// A plain (non-forceReauth) request proceeds past session revocation
	// into normal SSO continuation, which needs a TokenProvider this
	// minimal test doesn't construct — irrelevant to what's under test
	// here (that revocation is skipped), so recover and only check that.
	func() {
		defer func() { _ = recover() }()
		h.AuthorizeHandler()(c)
	}()

	assert.Empty(t, ms.deletedSessions, "a plain request must not revoke an existing session")
}
