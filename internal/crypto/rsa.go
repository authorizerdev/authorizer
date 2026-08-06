package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

// NewRSAKey to generate new RSA Key if env is not set
// returns key instance, private key string, public key string, jwk string, error
func NewRSAKey(algo, keyID string) (*rsa.PrivateKey, string, string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, "", "", "", err
	}

	privateKey, publicKey, err := AsRSAStr(key, &key.PublicKey)
	if err != nil {
		return nil, "", "", "", err
	}

	jwkPublicKey, err := GetPubJWK(algo, keyID, &key.PublicKey)
	if err != nil {
		return nil, "", "", "", err
	}

	return key, privateKey, publicKey, string(jwkPublicKey), err
}

// IsRSA checks if given string is valid RSA algo
func IsRSA(algo string) bool {
	switch algo {
	case "RS256", "RS384", "RS512":
		return true
	default:
		return false
	}
}

// ExportRsaPrivateKeyAsPemStr to get RSA private key as pem string
func ExportRsaPrivateKeyAsPemStr(privkey *rsa.PrivateKey) string {
	privkeyBytes := x509.MarshalPKCS1PrivateKey(privkey)
	privkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)
	return string(privkeyPem)
}

// ExportRsaPublicKeyAsPemStr to get RSA public key as pem string
func ExportRsaPublicKeyAsPemStr(pubkey *rsa.PublicKey) string {
	pubkeyBytes := x509.MarshalPKCS1PublicKey(pubkey)
	pubkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkeyBytes,
		},
	)

	return string(pubkeyPem)
}

// ParseRsaPrivateKeyFromPemStr parses an RSA private key from a PEM string,
// accepting BOTH encodings openssl emits:
//
//   - PKCS#1 — "BEGIN RSA PRIVATE KEY", what openssl 1.x genrsa produced.
//   - PKCS#8 — "BEGIN PRIVATE KEY", what openssl 3.x genrsa produces BY
//     DEFAULT, and therefore what anyone generating keys today gets.
//
// Only PKCS#1 was accepted before, so a key from a current openssl failed. The
// failure was late and silent: the server started, signup worked, and only
// token ISSUANCE failed — every login broke on an instance that looked healthy.
func ParseRsaPrivateKeyFromPemStr(privPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	if priv, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return priv, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected an RSA private key, got %T", parsed)
	}
	return priv, nil
}

// ParseRsaPublicKeyFromPemStr parses an RSA public key from a PEM string,
// accepting BOTH encodings:
//
//   - PKCS#1 — "BEGIN RSA PUBLIC KEY".
//   - PKIX/SPKI — "BEGIN PUBLIC KEY", which is what `openssl rsa -pubout`
//     emits, i.e. the standard way to derive a public key.
//
// Only PKCS#1 was accepted before, which broke /.well-known/jwks.json for
// those deployments — relying parties could not verify tokens even when
// signing worked. Note the ECDSA path already used ParsePKIXPublicKey, so the
// two algorithms disagreed on the accepted format.
func ParseRsaPublicKeyFromPemStr(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected an RSA public key, got %T", parsed)
	}
	return pub, nil
}

// AsRSAStr returns private, public key string or error
func AsRSAStr(privateKey *rsa.PrivateKey, publickKey *rsa.PublicKey) (string, string, error) {
	// Export the keys to pem string
	privPem := ExportRsaPrivateKeyAsPemStr(privateKey)
	pubPem := ExportRsaPublicKeyAsPemStr(publickKey)

	// Import the keys from pem string
	privParsed, err := ParseRsaPrivateKeyFromPemStr(privPem)
	if err != nil {
		return "", "", err
	}
	pubParsed, err := ParseRsaPublicKeyFromPemStr(pubPem)
	if err != nil {
		return "", "", err
	}

	// Export the newly imported keys
	privParsedPem := ExportRsaPrivateKeyAsPemStr(privParsed)
	pubParsedPem := ExportRsaPublicKeyAsPemStr(pubParsed)

	return privParsedPem, pubParsedPem, nil
}

func EncryptRSA(message string, key rsa.PublicKey) (string, error) {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, &key, []byte(message), label)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptRSA(cipherText string, privateKey rsa.PrivateKey) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, &privateKey, ct, label)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
