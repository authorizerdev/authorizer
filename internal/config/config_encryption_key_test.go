package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncryptionKeyFallsBackToJWTSecret pins the backwards-compatible path:
// an HS* install that never set --encryption-key must keep decrypting TOTP
// seeds written by earlier releases, which used JWTSecret.
func TestEncryptionKeyFallsBackToJWTSecret(t *testing.T) {
	c := &Config{JWTType: "HS256", JWTSecret: "the-jwt-secret"}
	c.Finalize()

	assert.Equal(t, "the-jwt-secret", c.EncryptionKey,
		"an unset encryption key must fall back to JWTSecret or existing TOTP enrollments break")
	require.NoError(t, c.ValidateEncryptionKey())
}

// TestExplicitEncryptionKeyWins pins that a dedicated key is not clobbered by
// the fallback — this is what lets an operator rotate JWTSecret without
// making every enrolled authenticator undecryptable.
func TestExplicitEncryptionKeyWins(t *testing.T) {
	c := &Config{JWTType: "HS256", JWTSecret: "the-jwt-secret", EncryptionKey: "a-separate-key"}
	c.Finalize()

	assert.Equal(t, "a-separate-key", c.EncryptionKey)
	require.NoError(t, c.ValidateEncryptionKey())
}

// TestAsymmetricJWTWithoutEncryptionKeyIsRejected is the regression test for
// the real gap: JWTSecret is only required for HMAC, so an RS*/ES* install
// leaves it empty and the fallback lands on "". HKDF over empty keying
// material yields a fixed, publicly computable AES key, so TOTP seeds would be
// stored unprotected while still looking encrypted. Startup must refuse.
func TestAsymmetricJWTWithoutEncryptionKeyIsRejected(t *testing.T) {
	c := &Config{
		JWTType:       "RS256",
		JWTPrivateKey: "-----BEGIN RSA PRIVATE KEY-----",
		JWTPublicKey:  "-----BEGIN PUBLIC KEY-----",
	}
	c.Finalize()

	require.Empty(t, c.EncryptionKey, "precondition: no JWTSecret to fall back to")
	err := c.ValidateEncryptionKey()
	require.Error(t, err, "an empty encryption key must not silently reach the cipher")
	assert.Contains(t, err.Error(), "--encryption-key is required")
}

// TestAsymmetricJWTWithEncryptionKeyStarts pins the documented remedy.
func TestAsymmetricJWTWithEncryptionKeyStarts(t *testing.T) {
	c := &Config{
		JWTType:       "RS256",
		JWTPrivateKey: "-----BEGIN RSA PRIVATE KEY-----",
		JWTPublicKey:  "-----BEGIN PUBLIC KEY-----",
		EncryptionKey: "a-strong-random-value",
	}
	c.Finalize()

	require.NoError(t, c.ValidateEncryptionKey())
}

// TestEncryptionKeyStillRequiredWhenTOTPDisabled pins that the check is NOT
// scoped to TOTP. The same key HMACs the OTP digests used by password reset and
// email/SMS verification, which are written regardless of TOTP, so turning TOTP
// off must not silently re-open the empty-key hole.
func TestEncryptionKeyStillRequiredWhenTOTPDisabled(t *testing.T) {
	c := &Config{JWTType: "RS256", DisableTOTPLogin: true}
	c.Finalize()

	err := c.ValidateEncryptionKey()
	require.Error(t, err, "password-reset OTPs are hashed with this key even when TOTP is off")
	assert.Contains(t, err.Error(), "--encryption-key is required")
}
