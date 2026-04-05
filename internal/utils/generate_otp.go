package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateOTP to generate random 6 digit otp
func GenerateOTP() (string, error) {
	code := ""
	codeLength := 6
	charSet := "ABCDEFGHJKLMNPQRSTUVWXYZ123456789"
	charSetLength := big.NewInt(int64(len(charSet)))
	for i := 0; i < codeLength; i++ {
		index, err := rand.Int(rand.Reader, charSetLength)
		if err != nil {
			return "", fmt.Errorf("failed to generate secure random number: %w", err)
		}
		code += string(charSet[index.Int64()])
	}

	return code, nil
}
