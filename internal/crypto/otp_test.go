package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashOTP_RoundTrip(t *testing.T) {
	const key = "test-jwt-secret"
	const plain = "123456"

	hashed := HashOTP(plain, key)

	// Hex digest of HMAC-SHA256 is always 64 chars (32 bytes * 2)
	require.Len(t, hashed, 64)
	// And must not equal the plaintext
	assert.NotEqual(t, plain, hashed)

	// Verify with the original plaintext succeeds
	assert.True(t, VerifyOTPHash(plain, hashed, key))
	// Verifying with the digest itself MUST fail — otherwise the digest
	// becomes a usable credential for anyone with DB read access.
	assert.False(t, VerifyOTPHash(hashed, hashed, key))
}

func TestHashOTP_Deterministic(t *testing.T) {
	// Same plaintext + same key must produce the same digest. Otherwise
	// VerifyOTPHash could not work — there is no random nonce as in AES.
	const key = "k"
	const plain = "999000"

	a := HashOTP(plain, key)
	b := HashOTP(plain, key)
	assert.Equal(t, a, b)
}

func TestVerifyOTPHash_WrongPlaintext(t *testing.T) {
	const key = "k"
	stored := HashOTP("123456", key)

	assert.False(t, VerifyOTPHash("123457", stored, key))
	assert.False(t, VerifyOTPHash("000000", stored, key))
	assert.False(t, VerifyOTPHash("", stored, key))
}

func TestVerifyOTPHash_WrongKey(t *testing.T) {
	const plain = "123456"
	stored := HashOTP(plain, "key-a")

	// Same plaintext, different server key → must not verify. This is
	// what protects against cross-tenant or cross-deployment leakage.
	assert.False(t, VerifyOTPHash(plain, stored, "key-b"))
	assert.True(t, VerifyOTPHash(plain, stored, "key-a"))
}

func TestVerifyOTPHash_DifferentLengthDigest(t *testing.T) {
	// Constant-time compare must still return false for differing
	// lengths, not panic.
	assert.False(t, VerifyOTPHash("123456", "shortdigest", "k"))
	assert.False(t, VerifyOTPHash("123456", strings.Repeat("a", 65), "k"))
}

func TestHashOTP_NeverEqualsPlain(t *testing.T) {
	// Defence-in-depth: a test that fails loudly if someone ever
	// "optimises" HashOTP into a no-op.
	for _, plain := range []string{"123456", "000000", "9", "aaaaaa"} {
		assert.NotEqual(t, plain, HashOTP(plain, "k"))
	}
}
