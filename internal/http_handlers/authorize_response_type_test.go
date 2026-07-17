package http_handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/config"
)

func authorizeCtx(query string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/authorize?"+query, nil)
	return c, rec
}

// RFC 6749 §3.1.1: response_type is REQUIRED. A request that omits it must
// be rejected, not silently defaulted to an implicit-flow token grant the
// client never asked for (regression test for the oidcc-response-type-missing
// conformance failure).
func TestAuthorize_MissingResponseType_NoRedirectURI_ReturnsJSONError(t *testing.T) {
	logger := zerolog.Nop()
	h := &httpProvider{
		Config:       &config.Config{},
		Dependencies: Dependencies{Log: &logger},
	}

	c, rec := authorizeCtx("client_id=abc&state=xyz")
	h.AuthorizeHandler()(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_request")
	assert.Contains(t, rec.Body.String(), "response_type is required")
	assert.NotContains(t, rec.Body.String(), "access_token")
}

func TestAuthorize_MissingResponseType_WithRedirectURI_RedirectsWithError(t *testing.T) {
	logger := zerolog.Nop()
	h := &httpProvider{
		Config:       &config.Config{AllowedOrigins: []string{"*"}},
		Dependencies: Dependencies{Log: &logger},
	}

	c, rec := authorizeCtx("client_id=abc&state=xyz&redirect_uri=" + "http%3A%2F%2Fexample.com%2Fapp%2Fcallback")
	h.AuthorizeHandler()(c)

	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "error=invalid_request")
	assert.NotContains(t, location, "access_token")
}
