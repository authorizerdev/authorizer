package token

import (
	"errors"

	"github.com/golang-jwt/jwt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/memorystore"
)

// SignJWTToken common util to sing jwt token
func SignJWTToken(claims jwt.MapClaims) (string, error) {
	jwtType, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtType)
	if err != nil {
		return "", err
	}
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
		jwtSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)
		if err != nil {
			return "", err
		}
		return t.SignedString([]byte(jwtSecret))
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		jwtPrivateKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtPrivateKey)
		if err != nil {
			return "", err
		}
		key, err := crypto.ParseRsaPrivateKeyFromPemStr(jwtPrivateKey)
		if err != nil {
			return "", err
		}
		return t.SignedString(key)
	case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		jwtPrivateKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtPrivateKey)
		if err != nil {
			return "", err
		}
		key, err := crypto.ParseEcdsaPrivateKeyFromPemStr(jwtPrivateKey)
		if err != nil {
			return "", err
		}

		return t.SignedString(key)
	default:
		return "", errors.New("unsupported signing method")
	}
}

// ParseJWTToken common util to parse jwt token
func ParseJWTToken(token string) (jwt.MapClaims, error) {
	jwtType, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtType)
	if err != nil {
		return nil, err
	}
	signingMethod := jwt.GetSigningMethod(jwtType)

	var claims jwt.MapClaims

	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			jwtSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)
			if err != nil {
				return nil, err
			}
			return []byte(jwtSecret), nil
		})
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			jwtPublicKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtPublicKey)
			if err != nil {
				return nil, err
			}
			key, err := crypto.ParseRsaPublicKeyFromPemStr(jwtPublicKey)
			if err != nil {
				return nil, err
			}
			return key, nil
		})
	case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			jwtPublicKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyJwtPublicKey)
			if err != nil {
				return nil, err
			}
			key, err := crypto.ParseEcdsaPublicKeyFromPemStr(jwtPublicKey)
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

	return claims, nil
}

// ValidateJWTClaims common util to validate claims
func ValidateJWTClaims(claims jwt.MapClaims, hostname, nonce, subject string) (bool, error) {
	clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	if err != nil {
		return false, err
	}
	if claims["aud"] != clientID {
		return false, errors.New("invalid audience")
	}

	if claims["nonce"] != nonce {
		return false, errors.New("invalid nonce")
	}

	if claims["iss"] != hostname {
		return false, errors.New("invalid issuer")
	}

	if claims["sub"] != subject {
		return false, errors.New("invalid subject")
	}

	return true, nil
}

// ValidateJWTTokenWithoutNonce common util to validate claims without nonce
func ValidateJWTTokenWithoutNonce(claims jwt.MapClaims, hostname, subject string) (bool, error) {
	clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	if err != nil {
		return false, err
	}
	if claims["aud"] != clientID {
		return false, errors.New("invalid audience")
	}

	if claims["iss"] != hostname {
		return false, errors.New("invalid issuer")
	}

	if claims["sub"] != subject {
		return false, errors.New("invalid subject")
	}
	return true, nil
}
