package http_handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/config"
)

func authorizePostCtx(form url.Values) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/authorize", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c, rec
}

// RFC 6749 §3.1 / OIDC Core §3.1.2.1: the authorization endpoint MUST
// support POST as well as GET (regression test for the
// oidcc-ensure-post-request-succeeds conformance failure — the handler
// used to read parameters via gc.Query() only, so a POST body was
// silently ignored).
func TestAuthorize_POSTFormBody_ParamsAreRead(t *testing.T) {
	logger := zerolog.Nop()
	h := &httpProvider{
		Config:       &config.Config{},
		Dependencies: Dependencies{Log: &logger},
	}

	form := url.Values{
		"client_id":     {"abc"},
		"state":         {"xyz"},
		"response_type": {"not_a_real_response_type"},
	}
	c, rec := authorizePostCtx(form)
	h.AuthorizeHandler()(c)

	// If the POST body were ignored, response_type would read as empty and
	// the handler would reject with "response_type is required" instead.
	// Getting unsupported_response_type proves the value was read from body.
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "unsupported_response_type")
}

// OIDCC-3.1.2.6 (JAR / RFC 9101): a server that does not process request
// objects MUST reject a request/request_uri parameter with
// request_not_supported / request_uri_not_supported rather than silently
// ignoring it and falling through to a confusing generic validation error
// (regression test for the oidcc-unsigned-request-object-supported-
// correctly-or-rejected-as-unsupported conformance failure).
func TestAuthorize_RequestObjectParam_RejectedAsUnsupported(t *testing.T) {
	logger := zerolog.Nop()
	h := &httpProvider{
		Config:       &config.Config{AllowedOrigins: []string{"*"}},
		Dependencies: Dependencies{Log: &logger, StorageProvider: &redirectURIClientStore{}},
	}

	c, rec := authorizeRedirectCtx("client_id=abc&state=xyz&response_type=code&redirect_uri=" +
		"http%3A%2F%2Fexample.com%2Fapp%2Fcallback&request=some.jwt.value")
	h.AuthorizeHandler()(c)

	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "error=request_not_supported")
}

func TestAuthorize_RequestURIParam_RejectedAsUnsupported(t *testing.T) {
	logger := zerolog.Nop()
	h := &httpProvider{
		Config:       &config.Config{AllowedOrigins: []string{"*"}},
		Dependencies: Dependencies{Log: &logger, StorageProvider: &redirectURIClientStore{}},
	}

	c, rec := authorizeRedirectCtx("client_id=abc&state=xyz&response_type=code&redirect_uri=" +
		"http%3A%2F%2Fexample.com%2Fapp%2Fcallback&request_uri=https%3A%2F%2Fexample.com%2Frequest.jwt")
	h.AuthorizeHandler()(c)

	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "error=request_uri_not_supported")
}
