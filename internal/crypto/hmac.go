package crypto

import (
	"crypto/rand"
	"encoding/hex"
)

// NewHMACKey returns a new cryptographically random key for HMAC signing.
func NewHMACKey(algo, keyID string) (string, string, error) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", err
	}
	key := hex.EncodeToString(keyBytes)
	jwkPublicKey, err := GetPubJWK(algo, keyID, []byte(key))
	if err != nil {
		return "", "", err
	}
	return key, string(jwkPublicKey), nil
}

// IsHMACValid checks if given string is valid HMCA algo
func IsHMACA(algo string) bool {
	switch algo {
	case "HS256", "HS384", "HS512":
		return true
	default:
		return false
	}
}
