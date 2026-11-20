package crypto

import (
	stdcrypto "crypto"
	"crypto/x509"
	"encoding/base64"

	"github.com/go-jose/go-jose/v4"
	"golang.org/x/crypto/bcrypt"
)

// KeyID returns the `kid` for a public signing key: its RFC 7638 JWK
// thumbprint (SHA-256, base64url). Derived from the key itself, so the same
// key keeps the same kid wherever it is published — primary or secondary —
// and a token signed before a rotation still names a key the JWKS lists.
// Signing (token.signJWTToken) and publishing (JWKsHandler) both use it, so
// the token header and the JWKS entry always agree.
func KeyID(publicKey any) (string, error) {
	thumbprint, err := (&jose.JSONWebKey{Key: publicKey}).Thumbprint(stdcrypto.SHA256)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(thumbprint), nil
}

// GetPubJWK returns JWK for given keys
func GetPubJWK(algo, keyID string, publicKey interface{}) (string, error) {
	jwk := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Algorithm:                   algo,
				Key:                         publicKey,
				Use:                         "sig",
				KeyID:                       keyID,
				Certificates:                []*x509.Certificate{},
				CertificateThumbprintSHA1:   []uint8{},
				CertificateThumbprintSHA256: []uint8{},
			},
		},
	}
	jwkPublicKey, err := jwk.Keys[0].MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(jwkPublicKey), nil
}

// // EncryptEnvData is used to encrypt the env data
// TODO: remove this function if not needed
// func EncryptEnvData(data map[string]interface{}) (string, error) {
// 	jsonBytes, err := json.Marshal(data)
// 	if err != nil {
// 		return "", err
// 	}

// 	storeData, err := memorystore.Provider.GetEnvStore()
// 	if err != nil {
// 		return "", err
// 	}

// 	err = json.Unmarshal(jsonBytes, &storeData)
// 	if err != nil {
// 		return "", err
// 	}

// 	configData, err := json.Marshal(storeData)
// 	if err != nil {
// 		return "", err
// 	}

// 	encryptedConfig, err := EncryptAESEnv(configData)
// 	if err != nil {
// 		return "", err
// 	}

// 	return EncryptB64(string(encryptedConfig)), nil
// }

// PasswordHashCost is the bcrypt cost for newly written password hashes.
//
// Raised from bcrypt.DefaultCost (10), which is below current guidance. This is
// write-side ONLY and needs no migration: bcrypt stores the cost inside the
// hash string, and CompareHashAndPassword reads it from there rather than from
// this constant. Existing cost-10 hashes therefore keep verifying at cost 10 —
// nobody is locked out — and only new signups and password resets get 12.
//
// The corollary is that raising this alone never upgrades anyone: a
// rehash-on-successful-login would be needed for that, and is deliberately not
// added here (it writes to the users table on every login, which wants its own
// change and its own load testing).
//
// Cost 12 is ~4x the CPU of cost 10 per verification. That is fine at login
// volume, and the per-account login lockout bounds how often an attacker can
// make us pay it.
const PasswordHashCost = 12

// EncryptPassword is used for encrypting password
func EncryptPassword(password string) (string, error) {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), PasswordHashCost)
	if err != nil {
		return "", err
	}

	return string(pw), nil
}
