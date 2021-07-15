package utils

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

func GetAuthToken(gc *gin.Context) (string, error) {
	token := ""
	cookie, err := gc.Request.Cookie(constants.COOKIE_NAME)
	if err != nil {
		// try to check in auth header for cookie
		log.Println("cookie not found checking headers")
		auth := gc.Request.Header.Get("Authorization")
		if auth == "" {
			return "", errors.New(`Unauthorized`)
		}

		token = strings.TrimPrefix(auth, "Bearer ")
	} else {
		token = cookie.Value
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
