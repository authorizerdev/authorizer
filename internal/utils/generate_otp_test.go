package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateOTP(t *testing.T) {
	t.Run("should return 6-character OTP", func(t *testing.T) {
		otp := GenerateOTP()
		assert.Len(t, otp, 6)
	})

	t.Run("should only contain valid charset characters", func(t *testing.T) {
		charset := "ABCDEFGHJKLMNPQRSTUVWXYZ123456789"
		for i := 0; i < 50; i++ {
			otp := GenerateOTP()
			for _, c := range otp {
				assert.Contains(t, charset, string(c), "OTP contains invalid character: %c", c)
			}
		}
	})

	t.Run("should generate unique OTPs", func(t *testing.T) {
		seen := make(map[string]bool)
		duplicates := 0
		for i := 0; i < 100; i++ {
			otp := GenerateOTP()
			if seen[otp] {
				duplicates++
			}
			seen[otp] = true
		}
		// With crypto/rand and 33^6 possibilities, duplicates should be extremely rare
		assert.LessOrEqual(t, duplicates, 2, "too many duplicate OTPs generated")
	})
}
