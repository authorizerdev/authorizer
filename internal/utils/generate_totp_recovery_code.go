package utils

import (
	"math/rand"
	"time"
)

// GenerateTOTPRecoveryCode generates a random 16-character recovery code.
func GenerateTOTPRecoveryCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 16)

	rand.Seed(time.Now().UnixNano())
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}

	return string(code)
}
