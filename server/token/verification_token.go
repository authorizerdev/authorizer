package token

import (
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/golang-jwt/jwt"
)

// CreateVerificationToken creates a verification JWT token
func CreateVerificationToken(email, tokenType, hostname, nonceHash, redirectURL string) (string, error) {
	clientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID)
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"iss":          hostname,
		"aud":          clientID,
		"sub":          email,
		"exp":          time.Now().Add(time.Minute * 30).Unix(),
		"iat":          time.Now().Unix(),
		"token_type":   tokenType,
		"nonce":        nonceHash,
		"redirect_uri": redirectURL,
	}

	return SignJWTToken(claims)
}
