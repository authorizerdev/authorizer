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
