package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptAES(t *testing.T) {
	key := "test-client-secret"
	plaintext := `{"sub":"user123","roles":["admin"],"nonce":"abc"}`

	encrypted, err := EncryptAES(key, plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := DecryptAES(key, encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptAES_DifferentNonces(t *testing.T) {
	key := "test-key"
	plaintext := "same-text"

	enc1, err := EncryptAES(key, plaintext)
	require.NoError(t, err)
	enc2, err := EncryptAES(key, plaintext)
	require.NoError(t, err)

	// Same plaintext should produce different ciphertexts due to random nonce
	assert.NotEqual(t, enc1, enc2)
}

func TestDecryptAES_TamperedCiphertext(t *testing.T) {
	key := "test-key"
	plaintext := "sensitive-data"

	encrypted, err := EncryptAES(key, plaintext)
	require.NoError(t, err)

	// Tamper with the ciphertext (flip a character in the middle)
	runes := []rune(encrypted)
	mid := len(runes) / 2
	if runes[mid] == 'A' {
		runes[mid] = 'B'
	} else {
		runes[mid] = 'A'
	}
	tampered := string(runes)

	_, err = DecryptAES(key, tampered)
	assert.Error(t, err, "GCM should detect tampered ciphertext")
}

func TestDecryptAES_WrongKey(t *testing.T) {
	encrypted, err := EncryptAES("correct-key", "secret")
	require.NoError(t, err)

	_, err = DecryptAES("wrong-key", encrypted)
	assert.Error(t, err, "Should fail with wrong key")
}

func TestDecryptAES_TooShort(t *testing.T) {
	_, err := DecryptAES("key", "dG9vc2hvcnQ")
	assert.Error(t, err)
}

func TestEncryptDecryptAES_ShortKey(t *testing.T) {
	// Even a short key should work properly via HKDF derivation
	key := "short"
	plaintext := "test data"

	encrypted, err := EncryptAES(key, plaintext)
	require.NoError(t, err)

	decrypted, err := DecryptAES(key, encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecryptAES_LongKey(t *testing.T) {
	key := strings.Repeat("a", 100)
	plaintext := "test data"

	encrypted, err := EncryptAES(key, plaintext)
	require.NoError(t, err)

	decrypted, err := DecryptAES(key, encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
