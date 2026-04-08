package token

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// verificationTokenTypes enumerates the short-lived internal tokens
// issued by CreateVerificationToken (signup, magic-link, forgot-password,
// invite, OTP). Per OIDC Core §2 the ID-token `sub` claim MUST be a
// stable user identifier, but these verification tokens are issued
// BEFORE the user record exists (signup/magic-link) or address the
// email channel directly (update-email). For those flows only, we
// permit `sub == email` as a legitimate fallback. All other token
// types (access, refresh, ID) require the canonical user ID.
var verificationTokenTypes = map[string]bool{
	constants.VerificationTypeBasicAuthSignup: true,
	constants.VerificationTypeMagicLinkLogin:  true,
	constants.VerificationTypeUpdateEmail:     true,
	constants.VerificationTypeForgotPassword:  true,
	constants.VerificationTypeInviteMember:    true,
	constants.VerificationTypeOTP:             true,
}

// secondaryKidSuffix is appended to the primary kid to derive the
// secondary key's `kid` header value. Mirrors the JWKS handler in
// internal/http_handlers/jwks.go which publishes the secondary key
// under "<ClientID>-secondary".
const secondaryKidSuffix = "-secondary"

// SignJWTToken common util to sign a jwt token. Sets the JOSE
// `kid` header to the configured ClientID so verifiers (including
// the Authorizer JWKS endpoint at /.well-known/jwks.json) can pick
// the right published JWK during a manual key rotation.
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
	// kid identifies the verification key for relying parties.
	// Authorizer publishes one JWK per active key with kid =
	// ClientID for the primary and ClientID + "-secondary" for
	// the optional rotation key.
	t.Header["kid"] = p.config.ClientID

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

// extractKidHeader returns the `kid` JOSE header from a compact JWS
// without verifying the signature. Used to pre-select the right
// verification key when both primary and secondary keys are configured.
// Returns an empty string when the header is absent, malformed, or not
// a string.
func extractKidHeader(token string) string {
	parser := jwt.NewParser()
	parsed, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil || parsed == nil {
		return ""
	}
	kid, _ := parsed.Header["kid"].(string)
	return kid
}

// parseJWTWithKidSelection chooses primary vs secondary verification
// based on the `kid` header and falls back to trying both keys when
// the header is missing or unknown (legacy tokens issued before C2 are
// kid-less). Used by both ParseJWTToken and ParseIDTokenHint.
//
// Behaviour:
//   - kid == ClientID                 → primary only.
//   - kid == ClientID + "-secondary"  → secondary only (if configured).
//   - kid missing/unknown             → primary first, then secondary.
//
// The fallback only fires for *signature* errors (jwt.ErrTokenSignatureInvalid).
// Non-signature errors (malformed, alg mismatch, claim error) short-circuit
// and the primary error is returned wrapped with %w so callers can use
// errors.Is.
func (p *provider) parseJWTWithKidSelection(token string, parseFn func(algo, secret, publicKey string) (jwt.MapClaims, error)) (jwt.MapClaims, error) {
	primaryKid := p.config.ClientID
	secondaryKid := p.config.ClientID + secondaryKidSuffix
	hasSecondary := p.config.JWTSecondaryType != ""
	kid := extractKidHeader(token)

	// kid explicitly identifies a key.
	if kid != "" {
		switch kid {
		case primaryKid:
			return parseFn(p.config.JWTType, p.config.JWTSecret, p.config.JWTPublicKey)
		case secondaryKid:
			if !hasSecondary {
				return nil, errors.New("token kid references secondary key but no secondary key is configured")
			}
			return parseFn(p.config.JWTSecondaryType, p.config.JWTSecondarySecret, p.config.JWTSecondaryPublicKey)
		}
		// Unknown kid — fall through to legacy try-both behaviour.
	}

	claims, err := parseFn(p.config.JWTType, p.config.JWTSecret, p.config.JWTPublicKey)
	if err == nil {
		return claims, nil
	}

	// Only fall back on signature failures so we never paper over
	// malformed tokens, alg-mismatch errors, or claim errors. The v4
	// ValidationError type implements Is() against ErrTokenSignatureInvalid.
	if !hasSecondary || !errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		return claims, fmt.Errorf("primary key verification failed: %w", err)
	}

	secondaryClaims, secondaryErr := parseFn(p.config.JWTSecondaryType, p.config.JWTSecondarySecret, p.config.JWTSecondaryPublicKey)
	if secondaryErr != nil {
		// Surface the primary signature error wrapped — secondary
		// is best-effort and the caller only needs to know that
		// verification failed.
		return claims, fmt.Errorf("primary key verification failed: %w", err)
	}
	if p.dependencies != nil && p.dependencies.Log != nil {
		// Useful rotation signal: a token in the wild was issued
		// under the previous (now-secondary) key. Token contents
		// are NOT logged.
		p.dependencies.Log.Debug().Msg("token verified by secondary key")
	}
	return secondaryClaims, nil
}

// ParseJWTToken common util to parse jwt token. On signature failure
// with the primary key, retries with the optional secondary key if one
// is configured — this supports manual key rotation. The signing key
// for NEW tokens is always the primary; the secondary is only used for
// verification so outstanding tokens issued with the previous key keep
// working during a rotation window. Honors the `kid` JOSE header to
// pick the right key directly when present.
func (p *provider) ParseJWTToken(token string) (jwt.MapClaims, error) {
	claims, err := p.parseJWTWithKidSelection(token, func(algo, secret, publicKey string) (jwt.MapClaims, error) {
		return p.parseJWTWithKey(token, algo, secret, publicKey)
	})
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
	claims["exp"] = int64(expFloat)

	// `iat` is OPTIONAL per RFC 7519 §4.1.6 and OIDC Core for
	// non-self-issued tokens. Normalize when present, otherwise
	// leave the claim untouched.
	if iatVal, ok := claims["iat"]; ok {
		iatFloat, ok := iatVal.(float64)
		if !ok {
			return claims, errors.New("invalid iat claim")
		}
		claims["iat"] = int64(iatFloat)
	}

	return claims, nil
}

// ValidateJWTClaims common util to validate claims
func (p *provider) ValidateJWTClaims(claims jwt.MapClaims, authTokenConfig *AuthTokenConfig) (bool, error) {
	if !AudienceMatches(claims["aud"], p.config.ClientID) {
		return false, errors.New("invalid audience")
	}

	if claims["nonce"] != authTokenConfig.Nonce {
		return false, errors.New("invalid nonce")
	}

	if claims["iss"] != authTokenConfig.HostName {
		return false, fmt.Errorf("invalid issuer iss[%s] != hostname[%s]", claims["iss"], authTokenConfig.HostName)
	}

	// OIDC Core §2: `sub` is a stable, never-reassigned identifier.
	// Accept ONLY the canonical user ID for OIDC-visible tokens
	// (access, refresh, ID). Verification tokens (signup, magic-link,
	// forgot-password, invite, OTP) are issued before a stable user
	// ID is available or address the email channel directly; for
	// those token_type values we still permit `sub == email`.
	if claims["sub"] != authTokenConfig.User.ID {
		tokenType, _ := claims["token_type"].(string)
		if !verificationTokenTypes[tokenType] || claims["sub"] != refs.StringValue(authTokenConfig.User.Email) {
			return false, errors.New("invalid subject")
		}
	}

	return true, nil
}

// ValidateJWTTokenWithoutNonce common util to validate claims without nonce
func (p *provider) ValidateJWTTokenWithoutNonce(claims jwt.MapClaims, authTokenConfig *AuthTokenConfig) (bool, error) {
	if !AudienceMatches(claims["aud"], p.config.ClientID) {
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
