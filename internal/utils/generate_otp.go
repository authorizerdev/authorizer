package utils

import (
	"crypto/rand"
	"math/big"
)

// GenerateOTP to generate random 6 digit otp
func GenerateOTP() string {
	code := ""
	codeLength := 6
	charSet := "ABCDEFGHJKLMNPQRSTUVWXYZ123456789"
	charSetLength := big.NewInt(int64(len(charSet)))
	for i := 0; i < codeLength; i++ {
		index, err := rand.Int(rand.Reader, charSetLength)
		if err != nil {
			panic("failed to generate secure random number: " + err.Error())
		}
		code += string(charSet[index.Int64()])
	}

	return code
}
