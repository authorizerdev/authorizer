package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// NewRSAKey to generate new RSA Key if env is not set
// returns key instance, private key string, public key string, jwk string, error
func NewRSAKey(algo, keyID string) (*rsa.PrivateKey, string, string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", "", "", err
	}

	privateKey, publicKey, err := AsRSAStr(key, &key.PublicKey)
	if err != nil {
		return nil, "", "", "", err
	}

	jwkPublicKey, err := GetPubJWK(algo, keyID, &key.PublicKey)
	if err != nil {
		return nil, "", "", "", err
	}

	return key, privateKey, publicKey, string(jwkPublicKey), err
}

// IsRSA checks if given string is valid RSA algo
func IsRSA(algo string) bool {
	switch algo {
	case "RS256", "RS384", "RS512":
		return true
	default:
		return false
	}
}

// ExportRsaPrivateKeyAsPemStr to get RSA private key as pem string
func ExportRsaPrivateKeyAsPemStr(privkey *rsa.PrivateKey) string {
	privkeyBytes := x509.MarshalPKCS1PrivateKey(privkey)
	privkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)
	return string(privkeyPem)
}

// ExportRsaPublicKeyAsPemStr to get RSA public key as pem string
func ExportRsaPublicKeyAsPemStr(pubkey *rsa.PublicKey) string {
	pubkeyBytes := x509.MarshalPKCS1PublicKey(pubkey)
	pubkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkeyBytes,
		},
	)

	return string(pubkeyPem)
}

// ParseRsaPrivateKeyFromPemStr to parse RSA private key from pem string
func ParseRsaPrivateKeyFromPemStr(privPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

// ParseRsaPublicKeyFromPemStr to parse RSA public key from pem string
func ParseRsaPublicKeyFromPemStr(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return pub, nil
}

// AsRSAStr returns private, public key string or error
func AsRSAStr(privateKey *rsa.PrivateKey, publickKey *rsa.PublicKey) (string, string, error) {
	// Export the keys to pem string
	privPem := ExportRsaPrivateKeyAsPemStr(privateKey)
	pubPem := ExportRsaPublicKeyAsPemStr(publickKey)

	// Import the keys from pem string
	privParsed, err := ParseRsaPrivateKeyFromPemStr(privPem)
	if err != nil {
		return "", "", err
	}
	pubParsed, err := ParseRsaPublicKeyFromPemStr(pubPem)
	if err != nil {
		return "", "", err
	}

	// Export the newly imported keys
	privParsedPem := ExportRsaPrivateKeyAsPemStr(privParsed)
	pubParsedPem := ExportRsaPublicKeyAsPemStr(pubParsed)

	return privParsedPem, pubParsedPem, nil
}
