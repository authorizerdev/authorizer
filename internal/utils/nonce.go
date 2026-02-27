package utils

import (
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

// GenerateNonce generates random nonce string and returns
// the nonce string, nonce hash, error
func GenerateNonce() (string, string, error) {
	nonce := uuid.New().String()
	nonceHash := crypto.EncryptB64(nonce)
	return nonce, nonceHash, nil
}

// EncryptNonce nonce string
func EncryptNonce(nonce string) (string, error) {
	nonceHash := crypto.EncryptB64(nonce)
	return nonceHash, nil
}

// DecryptNonce nonce string
func DecryptNonce(nonceHash string) (string, error) {
	nonce, err := crypto.DecryptB64(nonceHash)
	if err != nil {
		return "", err
	}
	return nonce, err
}
