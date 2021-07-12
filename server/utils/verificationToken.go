package utils

import (
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/yauthdev/yauth/server/constants"
)

type UserInfo struct {
	Email string `json:"email"`
}

type CustomClaimsExample struct {
	*jwt.StandardClaims
	TokenType string `json:"token_type"`
	UserInfo
}

// TODO convert tokenType to enum
func CreateVerificationToken(email string, tokenType string) (string, error) {
	t := jwt.New(jwt.GetSigningMethod(constants.JWT_TYPE))

	t.Claims = &CustomClaimsExample{
		&jwt.StandardClaims{

			ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
		},
		tokenType,
		UserInfo{Email: email},
	}

	return t.SignedString([]byte(constants.JWT_SECRET))
}

func VerifyVerificationToken(email string, tokenType string) (string, error) {
	t := jwt.New(jwt.GetSigningMethod(constants.JWT_TYPE))

	t.Claims = &CustomClaimsExample{
		&jwt.StandardClaims{

			ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
		},
		tokenType,
		UserInfo{Email: email},
	}

	return t.SignedString([]byte(constants.JWT_SECRET))
}
