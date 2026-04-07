package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// HashOTP returns the hex-encoded HMAC-SHA256 of the OTP value under the
// supplied server key. OTPs are short-lived (minutes) so we do not need a
// reversible primitive — only the verifier needs to be able to recompute
// the digest from a candidate value. This means an offline DB dump no
// longer reveals usable OTPs.
func HashOTP(otp, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(otp))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyOTPHash compares a candidate plaintext OTP against a stored HMAC
// digest in constant time.
func VerifyOTPHash(candidate, stored, key string) bool {
	expected := HashOTP(candidate, key)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(stored)) == 1
}
