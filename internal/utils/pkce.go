package utils

import (
	"crypto/rand"
	"crypto/sha256"
	b64 "encoding/base64"
	"fmt"
	"strings"
)

const (
	length = 32
)

// GenerateCodeChallenge creates PKCE-Code-Challenge
// and returns the verifier and challenge
func GenerateCodeChallenge() (verifier string, challenge string, err error) {
	randomBytes := make([]byte, length)
	if _, err = rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	verifier = strings.Trim(b64.URLEncoding.EncodeToString(randomBytes), "=")

	// Generate Challenge
	rawChallenge := sha256.New()
	rawChallenge.Write([]byte(verifier))
	challenge = strings.Trim(b64.URLEncoding.EncodeToString(rawChallenge.Sum(nil)), "=")

	return verifier, challenge, nil
}
