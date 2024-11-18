package crypto

import (
	"crypto/x509"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"

	"github.com/authorizerdev/authorizer/internal/config"
)

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

// GenerateJWKBasedOnEnv generates JWK based on env
// TODO update description
// make sure clientID, jwtType, jwtSecret / public & private key pair is set
// this is called while initializing app / when env is updated
func GenerateJWKBasedOnEnv(cfg config.Config) (string, error) {
	jwk := ""
	algo := cfg.JWTType
	clientID := cfg.ClientID
	jwtSecret := cfg.JWTSecret
	jwtPublicKey := cfg.JWTPublicKey
	var err error
	// check if jwt secret is provided
	if IsHMACA(algo) {
		jwk, err = GetPubJWK(algo, clientID, []byte(jwtSecret))
		if err != nil {
			return "", err
		}
	}

	if IsRSA(algo) {
		publicKeyInstance, err := ParseRsaPublicKeyFromPemStr(jwtPublicKey)
		if err != nil {
			return "", err
		}

		jwk, err = GetPubJWK(algo, clientID, publicKeyInstance)
		if err != nil {
			return "", err
		}
	}

	if IsECDSA(algo) {
		publicKeyInstance, err := ParseEcdsaPublicKeyFromPemStr(jwtPublicKey)
		if err != nil {
			return "", err
		}

		jwk, err = GetPubJWK(algo, clientID, publicKeyInstance)
		if err != nil {
			return "", err
		}
	}

	return jwk, nil
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

// EncryptPassword is used for encrypting password
func EncryptPassword(password string) (string, error) {
	pw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(pw), nil
}
