package token

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memory_store"
)

const adminSecretHeader = "x-authorizer-admin-secret"

// fakeSessionStore answers GetCache only — the one memory_store.Provider
// method ValidateAdminSession calls. Embedding the real interface (nil) for
// everything else keeps this to exactly the method the admin-session path
// exercises, the same pattern interceptors.stubTokenProvider uses for
// token.Provider.
type fakeSessionStore struct {
	memory_store.Provider
	sessions map[string]string
}

func (f *fakeSessionStore) GetCache(key string) (string, error) {
	return f.sessions[key], nil
}

// newSessionProvider reuses newProvider's config construction and wires in a
// live admin session for sessionID, so AdminAuthMode's cookie branch
// (ValidateAdminSession -> MemoryStoreProvider.GetCache) has something to hit.
func newSessionProvider(adminSecret, sessionID string) *provider {
	p := newProvider(adminSecret, false)
	p.dependencies = &Dependencies{
		MemoryStoreProvider: &fakeSessionStore{
			sessions: map[string]string{adminSessionCachePrefix + sessionID: "1"},
		},
	}
	return p
}

// newGinCtxWithAdminCookie builds a request carrying the admin session
// cookie, the way GetAdminAuthToken reads it. newGinCtx only sets headers, so
// this is a separate, equally small helper rather than an overload of it.
func newGinCtxWithAdminCookie(sessionID string) *gin.Context {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: constants.AdminCookieName, Value: sessionID})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

func TestAdminAuthMode_SharedSecretWhenHeaderMatches(t *testing.T) {
	p := newProvider("correct-secret", false)
	assert.Equal(t, constants.AuditAuthModeSharedSecret,
		p.AdminAuthMode(newGinCtx(adminSecretHeader, "correct-secret")))
}

func TestAdminAuthMode_EmptyWhenNotSuperAdmin(t *testing.T) {
	p := newProvider("correct-secret", false)
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "wrong-secret")))
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "")))
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx("", "")))
}

// IsSuperAdmin is re-expressed in terms of AdminAuthMode, so they must agree
// on every input. A divergence would mean a caller is admitted as super-admin
// while the audit record says they are not an admin at all.
func TestIsSuperAdmin_AgreesWithAdminAuthMode(t *testing.T) {
	p := newProvider("correct-secret", false)
	for _, secret := range []string{"correct-secret", "wrong-secret", ""} {
		gc := newGinCtx(adminSecretHeader, secret)
		assert.Equal(t, p.AdminAuthMode(gc) != "", p.IsSuperAdmin(gc),
			"disagreement for secret %q", secret)
	}
}

// An unconfigured secret must never authenticate, and must never be recorded
// as though it had.
func TestAdminAuthMode_EmptyAdminSecretNeverReportsSharedSecret(t *testing.T) {
	p := newProvider("", false)
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "")))
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "anything")))
}

// With header auth disabled, even the correct secret is not super admin.
func TestAdminAuthMode_RespectsDisableAdminHeaderAuth(t *testing.T) {
	p := newProvider("correct-secret", true)
	assert.Equal(t, "", p.AdminAuthMode(newGinCtx(adminSecretHeader, "correct-secret")))
}

// TestAdminAuthMode_AdminSessionWhenCookieValid covers the branch nothing
// else in this file reaches: a valid admin session cookie (dashboard login)
// must report AuditAuthModeAdminSession, not AuditAuthModeSharedSecret. Every
// other test here drives the header/secret path via newGinCtx, which never
// sets a cookie, so GetAdminAuthToken always failed and the session branch
// was never entered — if the two constants were swapped on this branch, or it
// returned the wrong one, nothing would fail.
func TestAdminAuthMode_AdminSessionWhenCookieValid(t *testing.T) {
	const sessionID = "valid-session-id"
	p := newSessionProvider("correct-secret", sessionID)
	gc := newGinCtxWithAdminCookie(sessionID)

	assert.Equal(t, constants.AuditAuthModeAdminSession, p.AdminAuthMode(gc))
	// Pin the predicate and the mode together on this branch too.
	assert.True(t, p.IsSuperAdmin(gc))
}
