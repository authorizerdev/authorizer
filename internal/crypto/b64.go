package crypto

import "encoding/base64"

// EncryptB64 encrypts data into base64 string
func EncryptB64(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

// DecryptB64 decrypts from base64 string to readable string
func DecryptB64(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
