package utils

import (
	"math/rand"
	"time"
)

func GenerateOTP() string {
	code := ""
	codeLength := 6
	charSet := "ABCDEFGHJKLMNPQRSTUVWXYZ123456789"
	charSetLength := int32(len(charSet))
	for i := 0; i < codeLength; i++ {
		index := randomNumber(0, charSetLength)
		code += string(charSet[index])
	}

	return code
}

func randomNumber(min, max int32) int32 {
	rand.Seed(time.Now().UnixNano())
	return min + int32(rand.Intn(int(max-min)))
}
