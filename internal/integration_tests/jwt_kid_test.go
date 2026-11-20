package integration_tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/crypto"
)

// JWKS consumers that pick the verification key by `kid` — SurrealDB's
// DEFINE ACCESS ... WITH JWT URL among them — reject a token whose header has
// none ("Missing token header 'kid'"). Every asymmetric token must carry a kid
// that names a key in /.well-known/jwks.json.

func tokenHeader(t *testing.T, token string) map[string]any {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[0])
	require.NoError(t, err)
	var h map[string]any
	require.NoError(t, json.Unmarshal(raw, &h))
	return h
}

func jwksKids(t *testing.T, ts *testSetup) []string {
	t.Helper()
	router := gin.New()
	router.GET("/.well-known/jwks.json", ts.HttpProvider.JWKsHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Keys []map[string]any `json:"keys"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	kids := make([]string, 0, len(body.Keys))
	for _, k := range body.Keys {
		kid, _ := k["kid"].(string)
		kids = append(kids, kid)
	}
	return kids
}

func signTestToken(t *testing.T, ts *testSetup) string {
	t.Helper()
	tok, err := ts.TokenProvider.SignJWTToken(jwt.MapClaims{"sub": "kid-test"})
	require.NoError(t, err)
	return tok
}

func TestJWTHeaderKidMatchesJWKS(t *testing.T) {
	newKey := map[string]func(cfg *config.Config) (string, string){
		"RS256": func(cfg *config.Config) (string, string) {
			_, priv, pub, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
			require.NoError(t, err)
			return priv, pub
		},
		"ES256": func(cfg *config.Config) (string, string) {
			_, priv, pub, _, err := crypto.NewECDSAKey("ES256", cfg.ClientID)
			require.NoError(t, err)
			return priv, pub
		},
	}
	for algo, gen := range newKey {
		t.Run(algo, func(t *testing.T) {
			cfg := getTestConfig()
			cfg.JWTType = algo
			cfg.JWTPrivateKey, cfg.JWTPublicKey = gen(cfg)
			cfg.JWTSecret = ""
			ts := initTestSetup(t, cfg)

			kid, ok := tokenHeader(t, signTestToken(t, ts))["kid"].(string)
			require.True(t, ok, "an %s token must carry a kid header", algo)
			assert.NotEmpty(t, kid)
			assert.Equal(t, []string{kid}, jwksKids(t, ts), "the token's kid must name the published key")
		})
	}
}

// After a manual rotation (new key primary, old key secondary), a token signed
// before the rotation must still name a key the JWKS publishes. A kid derived
// from a fixed label — the client ID — would now point at the NEW key and fail
// verification in any kid-strict consumer until the token expired.
func TestJWTKidSurvivesKeyRotation(t *testing.T) {
	base := getTestConfig()
	_, oldPriv, oldPub, _, err := crypto.NewRSAKey("RS256", base.ClientID)
	require.NoError(t, err)
	_, newPriv, newPub, _, err := crypto.NewRSAKey("RS256", base.ClientID)
	require.NoError(t, err)

	before := getTestConfig()
	before.JWTType, before.JWTPrivateKey, before.JWTPublicKey, before.JWTSecret = "RS256", oldPriv, oldPub, ""
	oldKid := tokenHeader(t, signTestToken(t, initTestSetup(t, before)))["kid"]

	after := getTestConfig()
	after.JWTType, after.JWTPrivateKey, after.JWTPublicKey, after.JWTSecret = "RS256", newPriv, newPub, ""
	after.JWTSecondaryType, after.JWTSecondaryPrivateKey, after.JWTSecondaryPublicKey = "RS256", oldPriv, oldPub
	ts := initTestSetup(t, after)
	newKid := tokenHeader(t, signTestToken(t, ts))["kid"]

	kids := jwksKids(t, ts)
	assert.Len(t, kids, 2)
	assert.Contains(t, kids, oldKid, "a pre-rotation token must still name a published key")
	assert.Contains(t, kids, newKid)
	assert.NotEqual(t, oldKid, newKid)
}

// HMAC keys are never published, so there is no JWKS entry to name.
func TestJWTHMACHasNoKid(t *testing.T) {
	cfg := getTestConfig()
	cfg.JWTType = "HS256"
	cfg.JWTSecret = "test-secret-at-least-32-bytes-long!!"
	ts := initTestSetup(t, cfg)
	_, has := tokenHeader(t, signTestToken(t, ts))["kid"]
	assert.False(t, has)
	assert.Empty(t, jwksKids(t, ts))
}
