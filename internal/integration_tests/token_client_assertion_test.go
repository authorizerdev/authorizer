package integration_tests

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestClientAssertionTokenEndpoint exercises the RFC 7523 client_assertion paths
// at the /oauth/token boundary that do NOT require a JWKS network fetch (the
// SSRF guard blocks loopback, so the signature-verifying happy path is covered
// by the clientauth resolver unit tests). Here we assert the transport-level
// contract: the multiple-auth-method rule, unsupported assertion types, and an
// unknown issuer.
func TestClientAssertionTokenEndpoint(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signAssertion := func(iss string) string {
		now := time.Now()
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss": iss,
			"sub": "system:serviceaccount:prod:payments",
			"aud": ccTestAuthorizerURL + "/oauth/token",
			"iat": now.Unix(),
			"exp": now.Add(5 * time.Minute).Unix(),
		})
		tok.Header["kid"] = "kid-1"
		s, sErr := tok.SignedString(key)
		require.NoError(t, sErr)
		return s
	}

	t.Run("secret_and_assertion_rejected", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("client_secret", "some-secret")
		form.Set("client_assertion", signAssertion("https://issuer.example.com"))
		form.Set("client_assertion_type", constants.ClientAssertionTypeJWTBearer)

		w := postClientCredentials(router, form, nil)
		require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		assert.Equal(t, "invalid_request", decodeJSON(t, w)["error"], "presenting a secret AND an assertion is >1 method (RFC 6749 §2.3)")
	})

	t.Run("unsupported_assertion_type_rejected", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("client_assertion", signAssertion("https://issuer.example.com"))
		form.Set("client_assertion_type", constants.ClientAssertionTypeJWTSPIFFE)

		w := postClientCredentials(router, form, nil)
		require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		assert.Equal(t, "invalid_request", decodeJSON(t, w)["error"])
	})

	t.Run("unknown_issuer_rejected", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", constants.GrantTypeClientCredentials)
		form.Set("client_assertion", signAssertion("https://unregistered-issuer.example.com"))
		form.Set("client_assertion_type", constants.ClientAssertionTypeJWTBearer)

		w := postClientCredentials(router, form, nil)
		require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		assert.Equal(t, "invalid_client", decodeJSON(t, w)["error"], "an assertion from an unregistered issuer must fail closed")
	})
}
