package token

import (
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/golang-jwt/jwt"
)

// CreateVerificationToken creates a verification JWT token
func CreateVerificationToken(email, tokenType, hostname string) (string, error) {
	claims := jwt.MapClaims{
		"exp":          time.Now().Add(time.Minute * 30).Unix(),
		"iat":          time.Now().Unix(),
		"token_type":   tokenType,
		"email":        email,
		"host":         hostname,
		"redirect_url": envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAppURL),
	}

	return SignJWTToken(claims)
}
