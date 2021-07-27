package utils

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
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

func CreateAuthToken(user UserAuthInfo, tokenType enum.TokenType) (string, int64, error) {
	t := jwt.New(jwt.GetSigningMethod(constants.JWT_TYPE))
	expiryBound := time.Hour
	if tokenType == enum.RefreshToken {
		// expires in 1 year
		expiryBound = time.Hour * 8760
	}

	expiresAt := time.Now().Add(expiryBound).Unix()

	t.Claims = &UserAuthClaim{
		&jwt.StandardClaims{
			ExpiresAt: expiresAt,
		},
		tokenType.String(),
		user,
	}

	token, err := t.SignedString([]byte(constants.JWT_SECRET))
	if err != nil {
		return "", 0, err
	}
	return token, expiresAt, nil
}

func GetAuthToken(gc *gin.Context) (string, error) {
	token, err := GetCookie(gc)
	if err != nil || token == "" {
		// try to check in auth header for cookie
		log.Println("cookie not found checking headers")
		auth := gc.Request.Header.Get("Authorization")
		if auth == "" {
			return "", fmt.Errorf(`Unauthorized`)
		}

		token = strings.TrimPrefix(auth, "Bearer ")
	}
	return token, nil
}

func VerifyAuthToken(token string) (*UserAuthClaim, error) {
	claims := &UserAuthClaim{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(constants.JWT_SECRET), nil
	})
	if err != nil {
		return claims, err
	}

	return claims, nil
}
