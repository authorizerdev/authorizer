package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

func TestJWKSPublishesSinglePrimaryKeyByDefault(t *testing.T) {
	cfg := getTestConfig()
	_, privateKey, publicKey, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)
	cfg.JWTType = "RS256"
	cfg.JWTPrivateKey = privateKey
	cfg.JWTPublicKey = publicKey
	cfg.JWTSecret = ""
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/.well-known/jwks.json", ts.HttpProvider.JWKsHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	keys, ok := body["keys"].([]interface{})
	require.True(t, ok)
	assert.Len(t, keys, 1, "JWKS MUST publish exactly one key when no secondary is configured")
}

func TestJWKSPublishesBothKeysWhenSecondaryConfigured(t *testing.T) {
	cfg := getTestConfig()
	// Primary RSA
	_, pPriv, pPub, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)
	cfg.JWTType = "RS256"
	cfg.JWTPrivateKey = pPriv
	cfg.JWTPublicKey = pPub
	cfg.JWTSecret = ""

	// Secondary RSA
	_, sPriv, sPub, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)
	cfg.JWTSecondaryType = "RS256"
	cfg.JWTSecondaryPrivateKey = sPriv
	cfg.JWTSecondaryPublicKey = sPub

	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/.well-known/jwks.json", ts.HttpProvider.JWKsHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	keys, ok := body["keys"].([]interface{})
	require.True(t, ok)
	require.Len(t, keys, 2, "JWKS MUST publish both keys when secondary is configured")

	kid0, _ := keys[0].(map[string]interface{})["kid"].(string)
	kid1, _ := keys[1].(map[string]interface{})["kid"].(string)
	assert.NotEqual(t, kid0, kid1, "primary and secondary keys MUST have distinct kids")
}

func TestJWKSSecondaryHMACIsNotExposed(t *testing.T) {
	cfg := getTestConfig() // default HS256
	// Configure a secondary HMAC - it must NOT be published.
	cfg.JWTSecondaryType = "HS256"
	cfg.JWTSecondarySecret = "some-secret"
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/.well-known/jwks.json", ts.HttpProvider.JWKsHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	keys, ok := body["keys"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, keys, "HMAC-only + HMAC secondary -> keys array MUST be empty")
}
