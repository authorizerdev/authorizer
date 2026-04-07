package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptTOTPSecret_RoundTrip(t *testing.T) {
	const key = "test-jwt-secret"
	const plain = "JBSWY3DPEHPK3PXP" // canonical RFC 6238 base32 example secret

	ct, err := EncryptTOTPSecret(plain, key)
	require.NoError(t, err)

	// Stored value must NOT equal the plaintext, must be marked with the
	// versioned prefix, and must round-trip back to the original.
	assert.NotEqual(t, plain, ct)
	assert.True(t, strings.HasPrefix(ct, TOTPCipherPrefix))
	assert.True(t, IsEncryptedTOTPSecret(ct))

	plainBack, err := DecryptTOTPSecret(ct, key)
	require.NoError(t, err)
	assert.Equal(t, plain, plainBack)
}

func TestEncryptTOTPSecret_NonceRandomness(t *testing.T) {
	// AES-GCM is non-deterministic — same input must produce different
	// ciphertext on each call (different nonce). Otherwise an attacker
	// can detect identical secrets across users.
	const key = "k"
	const plain = "JBSWY3DPEHPK3PXP"

	a, err := EncryptTOTPSecret(plain, key)
	require.NoError(t, err)
	b, err := EncryptTOTPSecret(plain, key)
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
	// Both must still decrypt to the same plaintext.
	pa, err := DecryptTOTPSecret(a, key)
	require.NoError(t, err)
	pb, err := DecryptTOTPSecret(b, key)
	require.NoError(t, err)
	assert.Equal(t, plain, pa)
	assert.Equal(t, plain, pb)
}

func TestDecryptTOTPSecret_LegacyRowReturnsSentinelError(t *testing.T) {
	// A row written by an older release will not have the enc:v1: prefix.
	// DecryptTOTPSecret is strict — it must return the sentinel error so
	// the totp authenticator can detect the legacy form and fall back to
	// using the raw stored value as a base32 secret (then migrate it on
	// the next successful Validate). The previous "silent passthrough"
	// API was a smell because callers couldn't tell the legacy case
	// apart from a real decryption.
	const legacyPlain = "JBSWY3DPEHPK3PXP"

	out, err := DecryptTOTPSecret(legacyPlain, "any-key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTOTPSecretNotEncrypted)
	assert.Equal(t, "", out)
	assert.False(t, IsEncryptedTOTPSecret(legacyPlain))
}

func TestDecryptTOTPSecret_TamperedCiphertext(t *testing.T) {
	const key = "k"
	const plain = "JBSWY3DPEHPK3PXP"

	ct, err := EncryptTOTPSecret(plain, key)
	require.NoError(t, err)

	// Flip a byte inside the AES-GCM payload (after the prefix). GCM is
	// authenticated, so any tamper must produce an error rather than
	// silently returning garbled bytes.
	body := strings.TrimPrefix(ct, TOTPCipherPrefix)
	tampered := TOTPCipherPrefix + flipChar(body)

	_, err = DecryptTOTPSecret(tampered, key)
	assert.Error(t, err)
}

func TestDecryptTOTPSecret_WrongKey(t *testing.T) {
	const plain = "JBSWY3DPEHPK3PXP"
	ct, err := EncryptTOTPSecret(plain, "key-a")
	require.NoError(t, err)

	_, err = DecryptTOTPSecret(ct, "key-b")
	assert.Error(t, err)
}

func TestEncryptTOTPSecret_EmptyInput(t *testing.T) {
	// An empty input is treated as "nothing to encrypt" — return empty
	// without trying to seal a zero-length plaintext. This matters for
	// upgrade paths where a half-initialised authenticator row exists.
	out, err := EncryptTOTPSecret("", "k")
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestIsEncryptedTOTPSecret(t *testing.T) {
	assert.False(t, IsEncryptedTOTPSecret(""))
	assert.False(t, IsEncryptedTOTPSecret("plain"))
	assert.False(t, IsEncryptedTOTPSecret("enc:v0:foo"))
	assert.True(t, IsEncryptedTOTPSecret("enc:v1:foo"))
}

// flipChar mutates one character in the middle of s so the result still has
// the same length but differs from the input — used to forge a tampered
// ciphertext for the GCM-authentication test.
func flipChar(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	mid := len(runes) / 2
	if runes[mid] == 'A' {
		runes[mid] = 'B'
	} else {
		runes[mid] = 'A'
	}
	return string(runes)
}
