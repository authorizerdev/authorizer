package crypto

import "github.com/google/uuid"

// NewHMAC key returns new key that can be used to ecnrypt data using HMAC algo
func NewHMACKey() string {
	key := uuid.New().String()
	return key
}

// IsHMACValid checks if given string is valid HMCA algo
func IsHMACA(algo string) bool {
	switch algo {
	case "HS256", "HS384", "HS512":
		return true
	default:
		return false
	}
}
