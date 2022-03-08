package token

import (
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/golang-jwt/jwt"
)

// CreateVerificationToken creates a verification JWT token
func CreateVerificationToken(email, tokenType, hostname, nonceHash, redirectURL string) (string, error) {
	claims := jwt.MapClaims{
		"iss":          hostname,
		"aud":          envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID),
		"sub":          email,
		"exp":          time.Now().Add(time.Minute * 30).Unix(),
		"iat":          time.Now().Unix(),
		"token_type":   tokenType,
		"nonce":        nonceHash,
		"redirect_uri": redirectURL,
	}

	return SignJWTToken(claims)
}
