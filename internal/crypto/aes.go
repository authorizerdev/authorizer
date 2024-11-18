package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

var bytes = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 0o5}

const (
	// Static key for encryption
	encryptionKey = "authorizerdev"
)

// EncryptAES method is to encrypt or hide any classified text
func EncryptAES(text string) (string, error) {
	key := []byte(encryptionKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plainText := []byte(text)
	cfb := cipher.NewCFBEncrypter(block, bytes)
	cipherText := make([]byte, len(plainText))
	cfb.XORKeyStream(cipherText, plainText)
	return EncryptB64(string(cipherText)), nil
}

// DecryptAES method is to extract back the encrypted text
func DecryptAES(text string) (string, error) {
	key := []byte(encryptionKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	cipherText, err := DecryptB64(text)
	if err != nil {
		return "", err
	}
	cfb := cipher.NewCFBDecrypter(block, bytes)
	plainText := make([]byte, len(cipherText))
	cfb.XORKeyStream(plainText, []byte(cipherText))
	return string(plainText), nil
}

// EncryptAESEnv encrypts data using AES algorithm
// kept for the backward compatibility of env data encryption
// TODO: Check if this is still needed
func EncryptAESEnv(text []byte) ([]byte, error) {
	var res []byte
	key := []byte(encryptionKey)
	c, err := aes.NewCipher(key)
	if err != nil {
		return res, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return res, err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return res, err
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return gcm.Seal(nonce, nonce, text, nil), nil
}

// DecryptAES decrypts data using AES algorithm
// Kept for the backward compatibility of env data decryption
// TODO: Check if this is still needed
func DecryptAESEnv(ciphertext []byte) ([]byte, error) {
	var res []byte
	key := []byte(encryptionKey)
	c, err := aes.NewCipher(key)
	if err != nil {
		return res, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return res, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return res, err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return res, err
	}

	return plaintext, nil
}
