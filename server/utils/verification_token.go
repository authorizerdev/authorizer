package utils

import (
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/golang-jwt/jwt"
)

// TODO see if we can move this to different service

// UserInfo is the user info that is stored in the JWT of verification request
type UserInfo struct {
	Email       string `json:"email"`
	Host        string `json:"host"`
	RedirectURL string `json:"redirect_url"`
}

// CustomClaim is the custom claim that is stored in the JWT of verification request
type CustomClaim struct {
	*jwt.StandardClaims
	TokenType string `json:"token_type"`
	UserInfo
}

// CreateVerificationToken creates a verification JWT token
func CreateVerificationToken(email string, tokenType string) (string, error) {
	t := jwt.New(jwt.GetSigningMethod(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyJwtType).(string)))

	t.Claims = &CustomClaim{
		&jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
		},
		tokenType,
		UserInfo{Email: email, Host: envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAuthorizerURL).(string), RedirectURL: envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAppURL).(string)},
	}

	return t.SignedString([]byte(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyJwtSecret).(string)))
}

// VerifyVerificationToken verifies the verification JWT token
func VerifyVerificationToken(token string) (*CustomClaim, error) {
	claims := &CustomClaim{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyJwtSecret).(string)), nil
	})
	if err != nil {
		return claims, err
	}

	return claims, nil
}
