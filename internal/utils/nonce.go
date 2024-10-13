package utils

import (
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

// GenerateNonce generats random nonce string and returns
// the nonce string, nonce hash, error
func GenerateNonce() (string, string, error) {
	nonce := uuid.New().String()
	nonceHash, err := crypto.EncryptAES(nonce)
	if err != nil {
		return "", "", err
	}
	return nonce, nonceHash, err
}

// EncryptNonce nonce string
func EncryptNonce(nonce string) (string, error) {
	nonceHash, err := crypto.EncryptAES(nonce)
	if err != nil {
		return "", err
	}
	return nonceHash, err
}

// DecryptNonce nonce string
func DecryptNonce(nonceHash string) (string, error) {
	nonce, err := crypto.DecryptAES(nonceHash)
	if err != nil {
		return "", err
	}
	return nonce, err
}
