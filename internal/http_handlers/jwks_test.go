package http_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/crypto"
)

// newJWKsTestProvider builds a minimal httpProvider suitable for unit
// testing JWKsHandler. It does not require any storage/memory_store
// dependencies because the handler only reads JWT config + logger.
func newJWKsTestProvider(t *testing.T, cfg *config.Config) *httpProvider {
	t.Helper()
	logger := zerolog.Nop()
	return &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log: &logger,
		},
	}
}

func doJWKsRequest(t *testing.T, h *httpProvider) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/.well-known/jwks.json", h.JWKsHandler())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	router.ServeHTTP(w, req)
	return w
}

// TestJWKsHandler_Success_ContentType verifies a happy-path RSA primary
// key is published as a valid JWK set with HTTP 200 + JSON content type.
func TestJWKsHandler_Success_ContentType(t *testing.T) {
	clientID := "test-client"
	_, _, publicKey, _, err := crypto.NewRSAKey("RS256", clientID)
	require.NoError(t, err)

	cfg := &config.Config{
		ClientID:     clientID,
		JWTType:      "RS256",
		JWTPublicKey: publicKey,
	}
	h := newJWKsTestProvider(t, cfg)

	w := doJWKsRequest(t, h)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	keys, ok := body["keys"].([]interface{})
	require.True(t, ok, "response MUST contain a keys array")
	require.Len(t, keys, 1, "exactly one primary key MUST be published")

	jwk, ok := keys[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "RSA", jwk["kty"])
	assert.NotEmpty(t, jwk["kid"])
	assert.NotEmpty(t, jwk["n"])
	assert.NotEmpty(t, jwk["e"])
}

// TestJWKsHandler_PrimaryKeyError_GenericResponse verifies that when
// primary key generation fails, the response body contains the generic
// OAuth2 server_error code and does NOT leak the underlying parser error.
func TestJWKsHandler_PrimaryKeyError_GenericResponse(t *testing.T) {
	cfg := &config.Config{
		ClientID:     "test-client",
		JWTType:      "RS256",
		JWTPublicKey: "this-is-not-a-valid-pem-block",
	}
	h := newJWKsTestProvider(t, cfg)

	w := doJWKsRequest(t, h)
	require.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "server_error", body["error"])
	assert.Equal(t, "failed to publish JWK set", body["error_description"])

	// MUST NOT leak the raw parser error message back to clients.
	rawBody := w.Body.String()
	assert.NotContains(t, strings.ToLower(rawBody), "pem")
	assert.NotContains(t, strings.ToLower(rawBody), "parse")
}

// TestJWKsHandler_SecondaryKeyError_StillReturnsPrimary verifies that a
// broken secondary key configuration is treated as degraded service:
// the primary key is still served and the response is HTTP 200.
func TestJWKsHandler_SecondaryKeyError_StillReturnsPrimary(t *testing.T) {
	clientID := "test-client"
	_, _, publicKey, _, err := crypto.NewRSAKey("RS256", clientID)
	require.NoError(t, err)

	cfg := &config.Config{
		ClientID:              clientID,
		JWTType:               "RS256",
		JWTPublicKey:          publicKey,
		JWTSecondaryType:      "RS256",
		JWTSecondaryPublicKey: "this-is-not-a-valid-pem-block",
	}
	h := newJWKsTestProvider(t, cfg)

	w := doJWKsRequest(t, h)
	require.Equal(t, http.StatusOK, w.Code, "broken secondary MUST NOT take down JWKS")

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	keys, ok := body["keys"].([]interface{})
	require.True(t, ok)
	require.Len(t, keys, 1, "only the primary key MUST be published when secondary fails")

	jwk, ok := keys[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "RSA", jwk["kty"])
}
