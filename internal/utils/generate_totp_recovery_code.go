package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateTOTPRecoveryCode generates a random 16-character recovery code.
func GenerateTOTPRecoveryCode() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 16)
	charSetLength := big.NewInt(int64(len(charset)))

	for i := range code {
		index, err := rand.Int(rand.Reader, charSetLength)
		if err != nil {
			return "", fmt.Errorf("failed to generate secure recovery code: %w", err)
		}
		code[i] = charset[index.Int64()]
	}

	return string(code), nil
}
