package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMetaFromGin_NilSafety(t *testing.T) {
	assert.Equal(t, RequestMetadata{}, MetaFromGin(nil))
	assert.Equal(t, RequestMetadata{}, MetaFromGin(&gin.Context{}))
}

func TestMetaFromGin_ExtractsRequestSignals(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://auth.example.com/x", nil)
	req.Host = "auth.example.com"
	req.Header.Set("Authorization", "Bearer abc")
	req.Header.Set("User-Agent", "AuthorizerTest/1.0")
	req.Header.Set("X-Forwarded-For", "10.1.2.3")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.AddCookie(&http.Cookie{Name: "session", Value: "s1"})

	gc, _ := gin.CreateTestContext(httptest.NewRecorder())
	gc.Request = req

	meta := MetaFromGin(gc)
	assert.Equal(t, "https://auth.example.com", meta.HostURL)
	assert.Equal(t, "10.1.2.3", meta.IPAddress)
	assert.Equal(t, "AuthorizerTest/1.0", meta.UserAgent)
	assert.Equal(t, "Bearer abc", meta.AuthorizationHeader)
	require.Len(t, meta.Cookies, 1)
	assert.Equal(t, "session", meta.Cookies[0].Name)
	assert.Same(t, req, meta.Request, "Request escape hatch must be the same pointer")
}

func TestApplyToGin_WritesCookies(t *testing.T) {
	w := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(w)
	gc.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	side := &ResponseSideEffects{}
	side.AddCookie(&http.Cookie{
		Name:     "authorizer_session",
		Value:    "abc",
		MaxAge:   60,
		Path:     "/",
		Domain:   "auth.example.com",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})
	side.AddCookie(&http.Cookie{
		Name:     "authorizer_session_domain",
		Value:    "abc",
		MaxAge:   60,
		Path:     "/",
		Domain:   ".example.com",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})
	ApplyToGin(gc, side)

	setCookies := w.Result().Header.Values("Set-Cookie")
	require.Len(t, setCookies, 2)
	assert.Contains(t, setCookies[0], "authorizer_session=abc")
	assert.Contains(t, setCookies[0], "Domain=auth.example.com")
	assert.Contains(t, setCookies[1], "Domain=example.com")
	for _, c := range setCookies {
		assert.Contains(t, c, "HttpOnly")
		assert.Contains(t, c, "Secure")
		assert.Contains(t, c, "SameSite=None")
	}
}

func TestApplyToGin_NilSafe(t *testing.T) {
	// nil receiver / nil gc must not panic.
	gc, _ := gin.CreateTestContext(httptest.NewRecorder())
	ApplyToGin(gc, nil)
	ApplyToGin(nil, &ResponseSideEffects{Cookies: []*http.Cookie{{Name: "x"}}})

	// nil cookie inside slice should be skipped.
	w := httptest.NewRecorder()
	gc2, _ := gin.CreateTestContext(w)
	gc2.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ApplyToGin(gc2, &ResponseSideEffects{Cookies: []*http.Cookie{nil, {Name: "ok", Value: "v"}}})
	assert.Len(t, w.Result().Header.Values("Set-Cookie"), 1)
}

func TestResponseSideEffects_AddCookieNilSafe(t *testing.T) {
	s := &ResponseSideEffects{}
	s.AddCookie(nil)
	assert.Empty(t, s.Cookies)
	s.AddCookie(&http.Cookie{Name: "x"})
	assert.Len(t, s.Cookies, 1)
}
