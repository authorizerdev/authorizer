package crypto

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
)

var bytes = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

// EncryptAES method is to encrypt or hide any classified text
func EncryptAES(text string) (string, error) {
	key := []byte(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyEncryptionKey))
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
	key := []byte(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyEncryptionKey))
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
