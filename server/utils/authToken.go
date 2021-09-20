package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// type UserAuthInfo struct {
// 	Email string `json:"email"`
// 	ID    string `json:"id"`
// }

type JWTCustomClaim map[string]interface{}

type UserAuthClaim struct {
	*jwt.StandardClaims
	*JWTCustomClaim `json:"authorizer"`
}

func CreateAuthToken(user db.User, tokenType enum.TokenType, role string) (string, int64, error) {
	t := jwt.New(jwt.GetSigningMethod(constants.JWT_TYPE))
	expiryBound := time.Hour
	if tokenType == enum.RefreshToken {
		// expires in 1 year
		expiryBound = time.Hour * 8760
	}

	expiresAt := time.Now().Add(expiryBound).Unix()

	customClaims := JWTCustomClaim{
		"token_type":             tokenType.String(),
		"email":                  user.Email,
		"id":                     user.ID,
		"allowed_roles":          strings.Split(user.Roles, ","),
		constants.JWT_ROLE_CLAIM: role,
	}

	t.Claims = &UserAuthClaim{
		&jwt.StandardClaims{
			ExpiresAt: expiresAt,
		},
		&customClaims,
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
			return "", fmt.Errorf(`unauthorized`)
		}

		token = strings.TrimPrefix(auth, "Bearer ")
	}
	return token, nil
}

func VerifyAuthToken(token string) (map[string]interface{}, error) {
	var res map[string]interface{}
	claims := &UserAuthClaim{}

	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(constants.JWT_SECRET), nil
	})
	if err != nil {
		return res, err
	}

	data, _ := json.Marshal(claims.JWTCustomClaim)
	json.Unmarshal(data, &res)
	res["exp"] = claims.ExpiresAt

	return res, nil
}
