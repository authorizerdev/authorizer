package crypto

import (
	"crypto/x509"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2"
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
