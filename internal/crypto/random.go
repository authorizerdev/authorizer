package crypto

import (
	"crypto/rand"
	"encoding/base64"
)

// NewRandomString returns a URL-safe, unpadded base64 encoding of numBytes of
// crypto/rand entropy — suitable for opaque handles (admin session ids, state
// nonces) that must be unguessable but carry no structure.
//
// numBytes is the ENTROPY, not the output length: base64 expands by ~4/3, so 32
// bytes yields a 43-character string.
func NewRandomString(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
