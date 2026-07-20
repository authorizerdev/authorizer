package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// samlIDPCertValidity is the lifetime of a generated SAML IdP signing cert.
// SAML signing certs are typically long-lived; rotation is an explicit operator
// action (see schemas.SAMLIDPKey), not driven by this expiry.
const samlIDPCertValidity = 5 * 365 * 24 * time.Hour

// NewSAMLSigningKeypair generates a fresh RSA-2048 private key and a self-signed
// X.509 certificate wrapping its public key, suitable for signing SAML assertions
// as an IdP. It returns the private key PEM (PKCS#1) and the certificate PEM.
//
// SAML XML-DSIG requires the X.509 certificate wrapper (SPs pin the
// <X509Certificate>); the raw-PEM JWT signing key cannot be reused because it
// carries no certificate. commonName is embedded as the cert subject/issuer CN
// (e.g. "Authorizer SAML IdP <org>") purely as a human-readable label.
func NewSAMLSigningKeypair(commonName string) (privateKeyPEM string, certPEM string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("generate rsa key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-1 * time.Minute),
		NotAfter:              now.Add(samlIDPCertValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", "", fmt.Errorf("create certificate: %w", err)
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	privateKeyPEM = ExportRsaPrivateKeyAsPemStr(key)
	return privateKeyPEM, certPEM, nil
}

// ParseCertificateFromPemStr parses an X.509 certificate from its PEM encoding.
func ParseCertificateFromPemStr(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}
