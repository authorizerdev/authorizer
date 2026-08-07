package crypto

import (
	"crypto/rand"
	"encoding/hex"
)

// NewHMACKey returns a new cryptographically random key for HMAC signing.
//
// The second return is deliberately empty. It used to be a JWK — but an HMAC
// key is SYMMETRIC, so its "public" JWK is {"kty":"oct","k":"<the signing
// secret>"}: the secret itself, wearing a public-key shape, stored in the
// public-key slot. The JWKS handler happens to filter to RSA/ECDSA today, so it
// was never served, but that is one `if` standing between a config field and
// full token-forgery capability for anyone who reads it. There is no legitimate
// consumer — a symmetric key has no public half to publish — so it is not
// generated at all.
func NewHMACKey(algo, keyID string) (string, string, error) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", err
	}
	return hex.EncodeToString(keyBytes), "", nil
}

// IsHMACA checks if given string is valid HMAC algo
func IsHMACA(algo string) bool {
	switch algo {
	case "HS256", "HS384", "HS512":
		return true
	default:
		return false
	}
}
