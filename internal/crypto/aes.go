package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// hkdfInfo is the fixed info string used for AES key derivation.
const hkdfInfo = "authorizer-aes-key"

// deriveAESKey derives a 32-byte AES key from the provided input keying
// material using HKDF-SHA256 with a fixed info string and no salt.
func deriveAESKey(ikm string) ([]byte, error) {
	reader := hkdf.New(sha256.New, []byte(ikm), nil, []byte(hkdfInfo))
	key := make([]byte, 32)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptAES encrypts plaintext using AES-256-GCM. The nonce is prepended to
// the ciphertext and the result is encoded as base64 RawURL.
func EncryptAES(key, text string) (string, error) {
	keyBytes, err := deriveAESKey(key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal appends the encrypted and authenticated ciphertext to nonce.
	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// DecryptAES decrypts a base64 RawURL-encoded AES-256-GCM ciphertext produced
// by EncryptAES. Returns an error if authentication fails or input is malformed.
func DecryptAES(key, encryptedText string) (string, error) {
	keyBytes, err := deriveAESKey(key)
	if err != nil {
		return "", err
	}

	data, err := base64.RawURLEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
