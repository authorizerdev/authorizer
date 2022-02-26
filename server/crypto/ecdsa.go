package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// NewECDSAKey to generate new ECDSA Key if env is not set
func NewECDSAKey() (*ecdsa.PrivateKey, string, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, "", "", err
	}

	privateKey, publicKey, err := AsECDSAStr(key, &key.PublicKey)
	if err != nil {
		return nil, "", "", err
	}

	return key, privateKey, publicKey, err
}

// IsECDSA checks if given string is valid ECDSA algo
func IsECDSA(algo string) bool {
	switch algo {
	case "ES256", "ES384", "ES512":
		return true
	default:
		return false
	}
}

// ExportEcdsaPrivateKeyAsPemStr to get ECDSA private key as pem string
func ExportEcdsaPrivateKeyAsPemStr(privkey *ecdsa.PrivateKey) (string, error) {
	privkeyBytes, err := x509.MarshalECPrivateKey(privkey)
	if err != nil {
		return "", err
	}
	privkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "ECDSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)
	return string(privkeyPem), nil
}

// ExportEcdsaPublicKeyAsPemStr to get ECDSA public key as pem string
func ExportEcdsaPublicKeyAsPemStr(pubkey *ecdsa.PublicKey) (string, error) {
	pubkeyBytes, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", err
	}
	pubkeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "ECDSA PUBLIC KEY",
			Bytes: pubkeyBytes,
		},
	)

	return string(pubkeyPem), nil
}

// ParseEcdsaPrivateKeyFromPemStr to parse ECDSA private key from pem string
func ParseEcdsaPrivateKeyFromPemStr(privPEM string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	priv, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

// ParseEcdsaPublicKeyFromPemStr to parse ECDSA public key from pem string
func ParseEcdsaPublicKeyFromPemStr(pubPEM string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *ecdsa.PublicKey:
		return pub, nil
	default:
		break // fall through
	}
	return nil, errors.New("Key type is not ECDSA")
}

// AsECDSAStr returns private, public key string or error
func AsECDSAStr(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (string, string, error) {
	// Export the keys to pem string
	privPem, err := ExportEcdsaPrivateKeyAsPemStr(privateKey)
	if err != nil {
		return "", "", err
	}
	pubPem, err := ExportEcdsaPublicKeyAsPemStr(publicKey)
	if err != nil {
		return "", "", err
	}

	// Import the keys from pem string
	privParsed, err := ParseEcdsaPrivateKeyFromPemStr(privPem)
	if err != nil {
		return "", "", err
	}
	pubParsed, err := ParseEcdsaPublicKeyFromPemStr(pubPem)
	if err != nil {
		return "", "", err
	}

	// Export the newly imported keys
	privParsedPem, err := ExportEcdsaPrivateKeyAsPemStr(privParsed)
	if err != nil {
		return "", "", err
	}
	pubParsedPem, err := ExportEcdsaPublicKeyAsPemStr(pubParsed)
	if err != nil {
		return "", "", err
	}

	return privParsedPem, pubParsedPem, nil
}
