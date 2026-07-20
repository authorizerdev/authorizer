package crypto

import (
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSAMLSigningKeypair(t *testing.T) {
	privPEM, certPEM, err := NewSAMLSigningKeypair("Authorizer SAML IdP test-org")
	require.NoError(t, err)
	require.NotEmpty(t, privPEM)
	require.NotEmpty(t, certPEM)

	// The private key must round-trip through the PKCS#1 PEM parser.
	priv, err := ParseRsaPrivateKeyFromPemStr(privPEM)
	require.NoError(t, err)
	require.IsType(t, &rsa.PrivateKey{}, priv)

	// The certificate must parse and wrap the same public key as the private key.
	cert, err := ParseCertificateFromPemStr(certPEM)
	require.NoError(t, err)
	assert.Equal(t, "Authorizer SAML IdP test-org", cert.Subject.CommonName)
	assert.True(t, cert.NotAfter.After(time.Now().Add(365*24*time.Hour)), "cert should be long-lived")

	certPub, ok := cert.PublicKey.(*rsa.PublicKey)
	require.True(t, ok)
	assert.Equal(t, priv.PublicKey.N, certPub.N, "cert public key must match the private key")
}

func TestNewSAMLSigningKeypair_Unique(t *testing.T) {
	_, cert1, err := NewSAMLSigningKeypair("org")
	require.NoError(t, err)
	_, cert2, err := NewSAMLSigningKeypair("org")
	require.NoError(t, err)
	assert.NotEqual(t, cert1, cert2, "each rotation must produce a distinct keypair")
}

func TestParseCertificateFromPemStr_Invalid(t *testing.T) {
	_, err := ParseCertificateFromPemStr("not a pem")
	assert.Error(t, err)
}
