package token

import (
	"errors"

	"github.com/golang-jwt/jwt"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

// SignJWTToken common util to sing jwt token
func (p *provider) SignJWTToken(jwtclaims jwt.MapClaims) (string, error) {
	signingMethod := jwt.GetSigningMethod(p.config.JWTType)
	if signingMethod == nil {
		return "", errors.New("unsupported signing method")
	}
	t := jwt.New(signingMethod)
	if t == nil {
		return "", errors.New("unsupported signing method")
	}
	t.Claims = jwtclaims

	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		return t.SignedString([]byte(p.config.JWTSecret))
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		key, err := crypto.ParseRsaPrivateKeyFromPemStr(p.config.JWTPublicKey)
		if err != nil {
			return "", err
		}
		return t.SignedString(key)
	case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		key, err := crypto.ParseEcdsaPrivateKeyFromPemStr(p.config.JWTPrivateKey)
		if err != nil {
			return "", err
		}

		return t.SignedString(key)
	default:
		return "", errors.New("unsupported signing method")
	}
}

// ParseJWTToken common util to parse jwt token
func (p *provider) ParseJWTToken(token string) (jwt.MapClaims, error) {
	signingMethod := jwt.GetSigningMethod(p.config.JWTType)

	var claims jwt.MapClaims
	var err error
	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(p.config.JWTSecret), nil
		})
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			key, err := crypto.ParseRsaPublicKeyFromPemStr(p.config.JWTPublicKey)
			if err != nil {
				return nil, err
			}
			return key, nil
		})
	case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		_, err = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			key, err := crypto.ParseEcdsaPublicKeyFromPemStr(p.config.JWTSecret)
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
func (p *provider) ValidateJWTClaims(claims jwt.MapClaims, authTokenConfig *AuthTokenConfig) (bool, error) {
	if claims["aud"] != p.config.ClientID {
		return false, errors.New("invalid audience")
	}

	if claims["nonce"] != authTokenConfig.Nonce {
		return false, errors.New("invalid nonce")
	}

	if claims["iss"] != authTokenConfig.HostName {
		return false, errors.New("invalid issuer")
	}

	if claims["sub"] != authTokenConfig.User.ID {
		return false, errors.New("invalid subject")
	}

	return true, nil
}

// ValidateJWTTokenWithoutNonce common util to validate claims without nonce
func (p *provider) ValidateJWTTokenWithoutNonce(claims jwt.MapClaims, authTokenConfig *AuthTokenConfig) (bool, error) {
	if claims["aud"] != p.config.ClientID {
		return false, errors.New("invalid audience")
	}

	if claims["iss"] != authTokenConfig.HostName {
		return false, errors.New("invalid issuer")
	}

	if claims["sub"] != authTokenConfig.User.ID {
		return false, errors.New("invalid subject")
	}
	return true, nil
}
