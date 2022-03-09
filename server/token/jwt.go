package token

import (
	"errors"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/golang-jwt/jwt"
)

// SignJWTToken common util to sing jwt token
func SignJWTToken(claims jwt.MapClaims) (string, error) {
	jwtType := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)
	signingMethod := jwt.GetSigningMethod(jwtType)
	if signingMethod == nil {
		return "", errors.New("unsupported signing method")
	}
	t := jwt.New(signingMethod)
	if t == nil {
		return "", errors.New("unsupported signing method")
	}
	t.Claims = claims

	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		return t.SignedString([]byte(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)))
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		key, err := crypto.ParseRsaPrivateKeyFromPemStr(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtPrivateKey))
		if err != nil {
			return "", err
		}
		return t.SignedString(key)
	case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		key, err := crypto.ParseEcdsaPrivateKeyFromPemStr(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtPrivateKey))
		if err != nil {
			return "", err
		}

		return t.SignedString(key)
	default:
		return "", errors.New("unsupported signing method")
	}
}

// ParseJWTToken common util to parse jwt token
func ParseJWTToken(token, hostname, nonce, subject string) (jwt.MapClaims, error) {
	jwtType := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)
	signingMethod := jwt.GetSigningMethod(jwtType)

	var err error
	var claims jwt.MapClaims

	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)), nil
		})
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			key, err := crypto.ParseRsaPublicKeyFromPemStr(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtPublicKey))
			if err != nil {
				return nil, err
			}
			return key, nil
		})
	case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			key, err := crypto.ParseEcdsaPublicKeyFromPemStr(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtPublicKey))
			if err != nil {
				return nil, err
			}
			return key, nil
		})
	default:
		err = errors.New("unsupported signing method")
	}
	if err != nil {
		return claims, err
	}

	// claim parses exp & iat into float 64 with e^10,
	// but we expect it to be int64
	// hence we need to assert interface and convert to int64
	intExp := int64(claims["exp"].(float64))
	intIat := int64(claims["iat"].(float64))
	claims["exp"] = intExp
	claims["iat"] = intIat

	if claims["aud"] != envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID) {
		return claims, errors.New("invalid audience")
	}

	if claims["nonce"] != nonce {
		return claims, errors.New("invalid nonce")
	}

	if claims["iss"] != hostname {
		return claims, errors.New("invalid issuer")
	}

	if claims["sub"] != subject {
		return claims, errors.New("invalid subject")
	}

	return claims, nil
}
