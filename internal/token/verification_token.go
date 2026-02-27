package token

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// CreateVerificationToken creates a verification JWT token
func (p *provider) CreateVerificationToken(authTokenConfig *AuthTokenConfig, redirectURL, tokenType string) (string, error) {
	claims := jwt.MapClaims{
		"iss":          authTokenConfig.HostName,
		"aud":          p.config.ClientID,
		"sub":          authTokenConfig.User.Email,
		"exp":          time.Now().Add(time.Minute * 30).Unix(),
		"iat":          time.Now().Unix(),
		"token_type":   tokenType,
		"nonce":        authTokenConfig.Nonce,
		"redirect_uri": redirectURL,
	}

	return p.SignJWTToken(claims)
}
