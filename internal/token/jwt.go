package token

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
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
		key, err := crypto.ParseRsaPrivateKeyFromPemStr(p.config.JWTPrivateKey)
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

// parseJWTWithKey is a helper shared by primary and secondary key
// verification. Returns the parsed claims or an error. No exp/iat
// normalization is performed; the caller handles that exactly once.
func (p *provider) parseJWTWithKey(token, algo, secret, publicKey string) (jwt.MapClaims, error) {
	signingMethod := jwt.GetSigningMethod(algo)
	if signingMethod == nil {
		return nil, errors.New("unsupported signing method")
	}

	var claims jwt.MapClaims
	var err error
	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		_, err = jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != signingMethod.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		})
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		_, err = jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != signingMethod.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			key, err := crypto.ParseRsaPublicKeyFromPemStr(publicKey)
			if err != nil {
				return nil, err
			}
			return key, nil
		})
	case jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		_, err = jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != signingMethod.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			key, err := crypto.ParseEcdsaPublicKeyFromPemStr(publicKey)
			if err != nil {
				return nil, err
			}
			return key, nil
		})
	default:
		err = errors.New("unsupported signing method")
	}
	return claims, err
}

// ParseJWTToken common util to parse jwt token. On signature failure
// with the primary key, retries with the optional secondary key if one
// is configured (Phase 3 manual key rotation support). The signing key
// for NEW tokens is always the primary; the secondary is only used for
// verification so outstanding tokens issued with the previous key keep
// working during a rotation window.
func (p *provider) ParseJWTToken(token string) (jwt.MapClaims, error) {
	claims, err := p.parseJWTWithKey(token, p.config.JWTType, p.config.JWTSecret, p.config.JWTPublicKey)
	if err != nil && p.config.JWTSecondaryType != "" {
		// Retry with the secondary key. Any non-nil error from the
		// primary path triggers the fallback — signature-invalid,
		// alg-mismatch, or parse error. If the secondary also fails,
		// return its error (the caller only cares that verification
		// failed, not which key failed).
		if secondaryClaims, secondaryErr := p.parseJWTWithKey(token, p.config.JWTSecondaryType, p.config.JWTSecondarySecret, p.config.JWTSecondaryPublicKey); secondaryErr == nil {
			claims = secondaryClaims
			err = nil
		}
	}
	if err != nil {
		return claims, err
	}

	// claim parses exp & iat into float64, but we expect int64.
	// Use safe type assertions to avoid panics on malformed tokens.
	expVal, ok := claims["exp"]
	if !ok {
		return claims, errors.New("missing exp claim")
	}
	expFloat, ok := expVal.(float64)
	if !ok {
		return claims, errors.New("invalid exp claim")
	}

	iatVal, ok := claims["iat"]
	if !ok {
		return claims, errors.New("missing iat claim")
	}
	iatFloat, ok := iatVal.(float64)
	if !ok {
		return claims, errors.New("invalid iat claim")
	}

	claims["exp"] = int64(expFloat)
	claims["iat"] = int64(iatFloat)

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
		return false, fmt.Errorf("invalid issuer iss[%s] != hostname[%s]", claims["iss"], authTokenConfig.HostName)
	}

	if claims["sub"] != authTokenConfig.User.ID && claims["sub"] != refs.StringValue(authTokenConfig.User.Email) {
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
		return false, fmt.Errorf("invalid issuer iss[%s] != hostname[%s]", claims["iss"], authTokenConfig.HostName)
	}

	if claims["sub"] != authTokenConfig.User.ID {
		return false, errors.New("invalid subject")
	}
	return true, nil
}
