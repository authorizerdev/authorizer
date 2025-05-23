package utils

import (
	"crypto/sha256"
	b64 "encoding/base64"
	"math/rand"
	"strings"
	"time"
)

const (
	length = 32
)

// GenerateCodeChallenge creates PKCE-Code-Challenge
// and returns the verifier and challenge
func GenerateCodeChallenge() (string, string) {
	// Generate Verifier
	randGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomBytes := make([]byte, length)
	for i := 0; i < length; i++ {
		randomBytes[i] = byte(randGenerator.Intn(255))
	}
	verifier := strings.Trim(b64.URLEncoding.EncodeToString(randomBytes), "=")

	// Generate Challenge
	rawChallenge := sha256.New()
	rawChallenge.Write([]byte(verifier))
	challenge := strings.Trim(b64.URLEncoding.EncodeToString(rawChallenge.Sum(nil)), "=")

	return verifier, challenge
}
