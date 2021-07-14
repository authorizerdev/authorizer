package utils

import (
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/yauthdev/yauth/server/constants"
	"github.com/yauthdev/yauth/server/enum"
)

type UserAuthInfo struct {
	Email string `json:"email"`
	ID    string `json:"id"`
}

type UserAuthClaim struct {
	*jwt.StandardClaims
	TokenType string `json:"token_type"`
	UserAuthInfo
}

func CreateAuthToken(user UserAuthInfo, tokenType enum.TokenType) (string, error) {
	t := jwt.New(jwt.GetSigningMethod(constants.JWT_TYPE))
	expiryBound := time.Hour
	if tokenType == enum.RefreshToken {
		// expires in 90 days
		expiryBound = time.Hour * 2160
	}

	t.Claims = &UserAuthClaim{
		&jwt.StandardClaims{
			ExpiresAt: time.Now().Add(expiryBound).Unix(),
		},
		tokenType.String(),
		user,
	}

	return t.SignedString([]byte(constants.JWT_SECRET))
}
