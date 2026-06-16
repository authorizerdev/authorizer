package token

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/config"
)

func newGinCtx(header, value string) *gin.Context {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if header != "" {
		req.Header.Set(header, value)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}

func newProvider(adminSecret string, disableHeaderAuth bool) *provider {
	return &provider{config: &config.Config{
		AdminSecret:            adminSecret,
		DisableAdminHeaderAuth: disableHeaderAuth,
	}}
}

func TestIsSuperAdmin_EmptyAdminSecretRejectsAllHeaderAuth(t *testing.T) {
	// AdminSecret not configured — header auth must be denied even for non-empty headers.
	p := newProvider("", false)
	assert.False(t, p.IsSuperAdmin(newGinCtx("x-authorizer-admin-secret", "anything")))
	assert.False(t, p.IsSuperAdmin(newGinCtx("x-authorizer-admin-secret", "")))
	assert.False(t, p.IsSuperAdmin(newGinCtx("", "")))
}

func TestIsSuperAdmin_WrongSecretRejected(t *testing.T) {
	p := newProvider("correct-secret", false)
	assert.False(t, p.IsSuperAdmin(newGinCtx("x-authorizer-admin-secret", "wrong")))
}

func TestIsSuperAdmin_CorrectSecretAccepted(t *testing.T) {
	p := newProvider("correct-secret", false)
	assert.True(t, p.IsSuperAdmin(newGinCtx("x-authorizer-admin-secret", "correct-secret")))
}

func TestIsSuperAdmin_HeaderAuthDisabledRejects(t *testing.T) {
	p := newProvider("correct-secret", true)
	assert.False(t, p.IsSuperAdmin(newGinCtx("x-authorizer-admin-secret", "correct-secret")))
}
