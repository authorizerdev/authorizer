package utils

import (
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/yauthdev/yauth/server/constants"
)

type UserInfo struct {
	Email string `json:"email"`
}

type CustomClaim struct {
	*jwt.StandardClaims
	TokenType string `json:"token_type"`
	UserInfo
}

// TODO convert tokenType to enum
func CreateVerificationToken(email string, tokenType string) (string, error) {
	t := jwt.New(jwt.GetSigningMethod(constants.JWT_TYPE))

	t.Claims = &CustomClaim{
		&jwt.StandardClaims{

			ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
		},
		tokenType,
		UserInfo{Email: email},
	}

	return t.SignedString([]byte(constants.JWT_SECRET))
}

func VerifyVerificationToken(token string) (*CustomClaim, error) {
	claims := &CustomClaim{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(constants.JWT_SECRET), nil
	})
	if err != nil {
		return claims, err
	}

	return claims, nil
}
