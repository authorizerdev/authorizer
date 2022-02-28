package crypto

import (
	"crypto/x509"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
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

// GenerateJWKBasedOnEnv generates JWK based on env
// make sure clientID, jwtType, jwtSecret / public & private key pair is set
// this is called while initializing app / when env is updated
func GenerateJWKBasedOnEnv() (string, error) {
	jwk := ""
	algo := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)
	clientID := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID)

	var err error
	// check if jwt secret is provided
	if IsHMACA(algo) {
		jwk, err = GetPubJWK(algo, clientID, []byte(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)))
		if err != nil {
			return "", err
		}
	}

	if IsRSA(algo) {
		publicKeyInstance, err := ParseRsaPublicKeyFromPemStr(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtPublicKey))
		if err != nil {
			return "", err
		}

		jwk, err = GetPubJWK(algo, clientID, publicKeyInstance)
		if err != nil {
			return "", err
		}
	}

	if IsECDSA(algo) {
		publicKeyInstance, err := ParseEcdsaPublicKeyFromPemStr(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtPublicKey))
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
