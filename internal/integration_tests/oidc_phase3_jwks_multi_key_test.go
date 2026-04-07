package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

// TestParseJWTTokenFallsBackToSecondaryKey verifies the Phase 3 manual
// rotation workflow: a token signed by the secondary key must still
// validate via ParseJWTToken when the primary key has changed.
func TestParseJWTTokenFallsBackToSecondaryKey(t *testing.T) {
	cfg := getTestConfig()
	// Primary RSA
	_, primaryPriv, primaryPub, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)
	// Secondary RSA (different key material)
	_, secondaryPriv, secondaryPub, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)

	cfg.JWTType = "RS256"
	cfg.JWTPrivateKey = primaryPriv
	cfg.JWTPublicKey = primaryPub
	cfg.JWTSecret = ""
	cfg.JWTSecondaryType = "RS256"
	cfg.JWTSecondaryPrivateKey = secondaryPriv
	cfg.JWTSecondaryPublicKey = secondaryPub

	ts := initTestSetup(t, cfg)

	// Build a JWT signed by the SECONDARY key — this is what an
	// outstanding token looks like after the operator swaps primary
	// and secondary during rotation.
	secondaryKey, err := crypto.ParseRsaPrivateKeyFromPemStr(secondaryPriv)
	require.NoError(t, err)
	claims := jwt.MapClaims{
		"iss":   "http://localhost",
		"aud":   cfg.ClientID,
		"sub":   "user-123",
		"nonce": "n",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(secondaryKey)
	require.NoError(t, err)

	// ParseJWTToken must verify successfully via the secondary-key fallback.
	parsed, err := ts.TokenProvider.ParseJWTToken(signed)
	require.NoError(t, err, "token signed by secondary key must verify via fallback")
	assert.Equal(t, "user-123", parsed["sub"])
}

// TestParseJWTTokenRejectsUnsignedGarbage ensures the secondary-key
// fallback doesn't accept arbitrary unsigned/garbage tokens.
func TestParseJWTTokenRejectsUnsignedGarbage(t *testing.T) {
	cfg := getTestConfig()
	_, primaryPriv, primaryPub, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)
	_, secondaryPriv, secondaryPub, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
	require.NoError(t, err)

	cfg.JWTType = "RS256"
	cfg.JWTPrivateKey = primaryPriv
	cfg.JWTPublicKey = primaryPub
	cfg.JWTSecret = ""
	cfg.JWTSecondaryType = "RS256"
	cfg.JWTSecondaryPrivateKey = secondaryPriv
	cfg.JWTSecondaryPublicKey = secondaryPub

	ts := initTestSetup(t, cfg)

	_, err = ts.TokenProvider.ParseJWTToken("not-a-jwt")
	assert.Error(t, err, "garbage token MUST fail even with secondary-key fallback")
}

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
