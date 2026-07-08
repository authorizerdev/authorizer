package http_handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const (
	ssoTestIssuer   = "https://idp.example.com"
	ssoTestClientID = "authorizer-rp-client"
	ssoTestNonce    = "nonce-abc-123"
	ssoTestKID      = "sso-key-1"
	ssoTestSubject  = "upstream-user-42"
)

func ssoGenKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return k
}

func ssoJWKS(pub *rsa.PublicKey, kid string) *jose.JSONWebKeySet {
	return &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: pub, KeyID: kid, Algorithm: "RS256", Use: "sig"}}}
}

func ssoSignRS256(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	require.NoError(t, err)
	return s
}

func ssoValidClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss":            ssoTestIssuer,
		"aud":            ssoTestClientID,
		"sub":            ssoTestSubject,
		"nonce":          ssoTestNonce,
		"email":          "user@corp.example.com",
		"email_verified": true,
		"iat":            now.Unix(),
		"exp":            now.Add(time.Hour).Unix(),
	}
}

func ssoFlowAndConn() (*ssoFlowState, *schemas.TrustedIssuer) {
	return &ssoFlowState{
			OrgID:          "org-1",
			ExpectedIssuer: ssoTestIssuer,
			Nonce:          ssoTestNonce,
		}, &schemas.TrustedIssuer{
			SSOClientID: ssoTestClientID,
		}
}

func TestSSOIDToken_ValidPasses(t *testing.T) {
	key := ssoGenKey(t)
	flow, conn := ssoFlowAndConn()
	claims, err := verifyIDTokenAgainstJWKS(flow, conn, ssoSignRS256(t, key, ssoTestKID, ssoValidClaims()), ssoJWKS(&key.PublicKey, ssoTestKID))
	require.NoError(t, err)
	assert.Equal(t, ssoTestSubject, claims["sub"])
}

// Mix-up defense (G3 / RFC 9207): an ID token whose iss differs from the
// dispatching connection's issuer must be rejected.
func TestSSOIDToken_MixupWrongIssuerRejected(t *testing.T) {
	key := ssoGenKey(t)
	flow, conn := ssoFlowAndConn()
	c := ssoValidClaims()
	c["iss"] = "https://attacker-idp.example.com"
	_, err := verifyIDTokenAgainstJWKS(flow, conn, ssoSignRS256(t, key, ssoTestKID, c), ssoJWKS(&key.PublicKey, ssoTestKID))
	require.Error(t, err)
}

func TestSSOIDToken_WrongAudienceRejected(t *testing.T) {
	key := ssoGenKey(t)
	flow, conn := ssoFlowAndConn()
	c := ssoValidClaims()
	c["aud"] = "some-other-client"
	_, err := verifyIDTokenAgainstJWKS(flow, conn, ssoSignRS256(t, key, ssoTestKID, c), ssoJWKS(&key.PublicKey, ssoTestKID))
	require.Error(t, err)
}

func TestSSOIDToken_WrongNonceRejected(t *testing.T) {
	key := ssoGenKey(t)
	flow, conn := ssoFlowAndConn()
	c := ssoValidClaims()
	c["nonce"] = "different-nonce"
	_, err := verifyIDTokenAgainstJWKS(flow, conn, ssoSignRS256(t, key, ssoTestKID, c), ssoJWKS(&key.PublicKey, ssoTestKID))
	require.Error(t, err)
}

func TestSSOIDToken_ExpiredRejected(t *testing.T) {
	key := ssoGenKey(t)
	flow, conn := ssoFlowAndConn()
	c := ssoValidClaims()
	c["exp"] = time.Now().Add(-2 * time.Hour).Unix()
	_, err := verifyIDTokenAgainstJWKS(flow, conn, ssoSignRS256(t, key, ssoTestKID, c), ssoJWKS(&key.PublicKey, ssoTestKID))
	require.Error(t, err)
}

// alg:none must be rejected by the asymmetric allow-list.
func TestSSOIDToken_AlgNoneRejected(t *testing.T) {
	flow, conn := ssoFlowAndConn()
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, ssoValidClaims())
	tok.Header["kid"] = ssoTestKID
	raw, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	key := ssoGenKey(t)
	_, err = verifyIDTokenAgainstJWKS(flow, conn, raw, ssoJWKS(&key.PublicKey, ssoTestKID))
	require.Error(t, err)
}

// OIDC Core §3.1.3.7 step 4: a multi-valued aud without a matching azp is rejected.
func TestSSOIDToken_MultiAudMissingAzpRejected(t *testing.T) {
	key := ssoGenKey(t)
	flow, conn := ssoFlowAndConn()
	c := ssoValidClaims()
	c["aud"] = []string{ssoTestClientID, "another-rp"}
	// no azp
	_, err := verifyIDTokenAgainstJWKS(flow, conn, ssoSignRS256(t, key, ssoTestKID, c), ssoJWKS(&key.PublicKey, ssoTestKID))
	require.Error(t, err)
}

// A multi-valued aud WITH azp == our client_id is accepted.
func TestSSOIDToken_MultiAudWithAzpPasses(t *testing.T) {
	key := ssoGenKey(t)
	flow, conn := ssoFlowAndConn()
	c := ssoValidClaims()
	c["aud"] = []string{ssoTestClientID, "another-rp"}
	c["azp"] = ssoTestClientID
	_, err := verifyIDTokenAgainstJWKS(flow, conn, ssoSignRS256(t, key, ssoTestKID, c), ssoJWKS(&key.PublicKey, ssoTestKID))
	require.NoError(t, err)
}

// A token signed by a key NOT in the JWKS (but claiming a known kid) must fail
// signature verification.
func TestSSOIDToken_WrongSignatureRejected(t *testing.T) {
	signingKey := ssoGenKey(t)
	jwksKey := ssoGenKey(t) // different key published in the JWKS
	flow, conn := ssoFlowAndConn()
	_, err := verifyIDTokenAgainstJWKS(flow, conn, ssoSignRS256(t, signingKey, ssoTestKID, ssoValidClaims()), ssoJWKS(&jwksKey.PublicKey, ssoTestKID))
	require.Error(t, err)
}
