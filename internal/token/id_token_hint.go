package token

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"

	"github.com/authorizerdev/authorizer/internal/crypto"
)

// ParseIDTokenHint parses an ID Token presented as the `id_token_hint`
// authentication or end-session parameter (OIDC Core §3.1.2.1, OIDC
// RP-Initiated Logout 1.0 §2). Per the spec, the OP SHOULD allow
// expired ID tokens as hints because the typical hint use-case is
// "this is who I think the user was when their session lapsed".
//
// This helper therefore verifies the signature only and skips
// time-based claim validation (`exp`, `nbf`, `iat`). Structural
// validation (alg-mismatch, malformed token) still applies. The same
// kid-aware key selection as ParseJWTToken is used so primary and
// secondary keys are honored during a rotation window.
//
// The returned MapClaims has `exp` / `iat` left as their parser-native
// float64 form (no normalization). Callers that need typed values
// should convert explicitly. The hint MUST NOT be trusted for
// authorization — only as a soft identifier hint.
func (p *provider) ParseIDTokenHint(token string) (jwt.MapClaims, error) {
	if token == "" {
		return nil, errors.New("empty id_token_hint")
	}
	return p.parseJWTWithKidSelection(token, func(algo, secret, publicKey string) (jwt.MapClaims, error) {
		return p.parseJWTHintWithKey(token, algo, secret, publicKey)
	})
}

// parseJWTHintWithKey verifies the signature of an id_token_hint
// against a single key, skipping time-based claim validation. Mirrors
// parseJWTWithKey but uses jwt.NewParser(jwt.WithoutClaimsValidation()).
func (p *provider) parseJWTHintWithKey(token, algo, secret, publicKey string) (jwt.MapClaims, error) {
	signingMethod := jwt.GetSigningMethod(algo)
	if signingMethod == nil {
		return nil, errors.New("unsupported signing method")
	}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	var claims jwt.MapClaims
	var err error
	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		_, err = parser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != signingMethod.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		})
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		_, err = parser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
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
		_, err = parser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
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
