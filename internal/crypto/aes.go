package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// var bytes = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 0o5}

// const (
// 	// Static key for encryption
// 	encryptionKey = "authorizerdev"
// )

// EncryptAES method is to encrypt or hide any classified text
func EncryptAES(key, text string) (string, error) {
	keyBytes := []byte(ensureHashKey(key))
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore, it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(text))

	// Encode the ciphertext to URL-safe base64 without padding
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// DecryptAES method is to extract back the encrypted text
func DecryptAES(key, encryptedText string) (string, error) {
	keyBytes := []byte(ensureHashKey(key))
	ciphertext, err := base64.RawURLEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

// ensureHashKey ensure the key is 32 bytes long
// if short it will append 0's to the key
// if long it will truncate the key
func ensureHashKey(key string) string {
	if len(key) < 32 {
		return key + string(bytes.Repeat([]byte{0}, 32-len(key)))
	}
	return key[:32]
}

// EncryptAESEnv encrypts data using AES algorithm
// kept for the backward compatibility of env data encryption
// TODO: Check if this is still needed
// func EncryptAESEnv(text []byte) ([]byte, error) {
// 	var res []byte
// 	key := []byte(encryptionKey)
// 	c, err := aes.NewCipher(key)
// 	if err != nil {
// 		return res, err
// 	}

// 	// gcm or Galois/Counter Mode, is a mode of operation
// 	// for symmetric key cryptographic block ciphers
// 	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
// 	gcm, err := cipher.NewGCM(c)
// 	if err != nil {
// 		return res, err
// 	}

// 	// creates a new byte array the size of the nonce
// 	// which must be passed to Seal
// 	nonce := make([]byte, gcm.NonceSize())
// 	// populates our nonce with a cryptographically secure
// 	// random sequence
// 	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
// 		return res, err
// 	}

// 	// here we encrypt our text using the Seal function
// 	// Seal encrypts and authenticates plaintext, authenticates the
// 	// additional data and appends the result to dst, returning the updated
// 	// slice. The nonce must be NonceSize() bytes long and unique for all
// 	// time, for a given key.
// 	return gcm.Seal(nonce, nonce, text, nil), nil
// }

// // DecryptAES decrypts data using AES algorithm
// // Kept for the backward compatibility of env data decryption
// // TODO: Check if this is still needed
// func DecryptAESEnv(ciphertext []byte) ([]byte, error) {
// 	var res []byte
// 	key := []byte(encryptionKey)
// 	c, err := aes.NewCipher(key)
// 	if err != nil {
// 		return res, err
// 	}

// 	gcm, err := cipher.NewGCM(c)
// 	if err != nil {
// 		return res, err
// 	}

// 	nonceSize := gcm.NonceSize()
// 	if len(ciphertext) < nonceSize {
// 		return res, err
// 	}

// 	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
// 	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
// 	if err != nil {
// 		return res, err
// 	}

// 	return plaintext, nil
// }
