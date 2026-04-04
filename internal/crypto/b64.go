package crypto

import "encoding/base64"

// EncodeB64 encodes data to a base64 string.
func EncodeB64(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

// DecodeB64 decodes a base64 string back to plaintext.
func DecodeB64(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// EncryptB64 is a deprecated alias for EncodeB64.
func EncryptB64(text string) string {
	return EncodeB64(text)
}

// DecryptB64 is a deprecated alias for DecodeB64.
func DecryptB64(s string) (string, error) {
	return DecodeB64(s)
}
