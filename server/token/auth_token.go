package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/robertkrimen/otto"
)

// JWTToken is a struct to hold JWT token and its expiration time
type JWTToken struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// Token object to hold the finger print and refresh token information
type Token struct {
	FingerPrint     string    `json:"fingerprint"`
	FingerPrintHash string    `json:"fingerprint_hash"`
	RefreshToken    *JWTToken `json:"refresh_token"`
	AccessToken     *JWTToken `json:"access_token"`
}

// CreateAuthToken creates a new auth token when userlogs in
func CreateAuthToken(user models.User, roles []string) (*Token, error) {
	fingerprint := uuid.NewString()
	fingerPrintHashBytes, err := utils.EncryptAES([]byte(fingerprint))
	if err != nil {
		return nil, err
	}
	refreshToken, refreshTokenExpiresAt, err := CreateRefreshToken(user, roles)
	if err != nil {
		return nil, err
	}

	accessToken, accessTokenExpiresAt, err := CreateAccessToken(user, roles)
	if err != nil {
		return nil, err
	}

	return &Token{
		FingerPrint:     fingerprint,
		FingerPrintHash: string(fingerPrintHashBytes),
		RefreshToken:    &JWTToken{Token: refreshToken, ExpiresAt: refreshTokenExpiresAt},
		AccessToken:     &JWTToken{Token: accessToken, ExpiresAt: accessTokenExpiresAt},
	}, nil
}

// CreateRefreshToken util to create JWT token
func CreateRefreshToken(user models.User, roles []string) (string, int64, error) {
	t := jwt.New(jwt.GetSigningMethod(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)))
	// expires in 1 year
	expiryBound := time.Hour * 8760
	expiresAt := time.Now().Add(expiryBound).Unix()

	customClaims := jwt.MapClaims{
		"exp":        expiresAt,
		"iat":        time.Now().Unix(),
		"token_type": constants.TokenTypeRefreshToken,
		"roles":      roles,
		"id":         user.ID,
	}

	t.Claims = customClaims
	token, err := t.SignedString([]byte(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)))
	if err != nil {
		return "", 0, err
	}
	return token, expiresAt, nil
}

// CreateAccessToken util to create JWT token, based on
// user information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT
func CreateAccessToken(user models.User, roles []string) (string, int64, error) {
	t := jwt.New(jwt.GetSigningMethod(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)))
	expiryBound := time.Minute * 30

	expiresAt := time.Now().Add(expiryBound).Unix()

	resUser := user.AsAPIUser()
	userBytes, _ := json.Marshal(&resUser)
	var userMap map[string]interface{}
	json.Unmarshal(userBytes, &userMap)

	claimKey := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtRoleClaim)
	customClaims := jwt.MapClaims{
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"token_type":    constants.TokenTypeAccessToken,
		"allowed_roles": strings.Split(user.Roles, ","),
		claimKey:        roles,
	}

	for k, v := range userMap {
		if k != "roles" {
			customClaims[k] = v
		}
	}

	// check for the extra access token script
	accessTokenScript := os.Getenv(constants.EnvKeyCustomAccessTokenScript)
	if accessTokenScript != "" {
		vm := otto.New()

		claimBytes, _ := json.Marshal(customClaims)
		vm.Run(fmt.Sprintf(`
			var user = %s;
			var tokenPayload = %s;
			var customFunction = %s;
			var functionRes = JSON.stringify(customFunction(user, tokenPayload));
		`, string(userBytes), string(claimBytes), accessTokenScript))

		val, err := vm.Get("functionRes")

		if err != nil {
			log.Println("error getting custom access token script:", err)
		} else {
			extraPayload := make(map[string]interface{})
			err = json.Unmarshal([]byte(fmt.Sprintf("%s", val)), &extraPayload)
			if err != nil {
				log.Println("error converting accessTokenScript response to map:", err)
			} else {
				for k, v := range extraPayload {
					customClaims[k] = v
				}
			}
		}
	}

	t.Claims = customClaims

	token, err := t.SignedString([]byte(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)))
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}

// GetAccessToken returns the access token from the request (either from header or cookie)
func GetAccessToken(gc *gin.Context) (string, error) {
	token, err := cookie.GetAccessTokenCookie(gc)
	if err != nil || token == "" {
		// try to check in auth header for cookie
		auth := gc.Request.Header.Get("Authorization")
		if auth == "" {
			return "", fmt.Errorf(`unauthorized`)
		}

		token = strings.TrimPrefix(auth, "Bearer ")

	}
	return token, nil
}

// GetRefreshToken returns the refresh token from cookie / request query url
func GetRefreshToken(gc *gin.Context) (string, error) {
	token, err := cookie.GetRefreshTokenCookie(gc)

	if err != nil || token == "" {
		return "", fmt.Errorf(`unauthorized`)
	}

	return token, nil
}

// GetFingerPrint returns the finger print from cookie
func GetFingerPrint(gc *gin.Context) (string, error) {
	fingerPrint, err := cookie.GetFingerPrintCookie(gc)
	if err != nil || fingerPrint == "" {
		return "", fmt.Errorf(`no finger print`)
	}
	return fingerPrint, nil
}

// VerifyJWTToken helps in verifying the JWT token
func VerifyJWTToken(token string) (map[string]interface{}, error) {
	var res map[string]interface{}
	claims := jwt.MapClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)), nil
	})
	if err != nil {
		return res, err
	}

	if !t.Valid {
		return res, fmt.Errorf(`invalid token`)
	}

	// claim parses exp & iat into float 64 with e^10,
	// but we expect it to be int64
	// hence we need to assert interface and convert to int64
	intExp := int64(claims["exp"].(float64))
	intIat := int64(claims["iat"].(float64))

	data, _ := json.Marshal(claims)
	json.Unmarshal(data, &res)
	res["exp"] = intExp
	res["iat"] = intIat

	return res, nil
}

func ValidateAccessToken(gc *gin.Context) (map[string]interface{}, error) {
	token, err := GetAccessToken(gc)
	if err != nil {
		return nil, err
	}

	claims, err := VerifyJWTToken(token)
	if err != nil {
		return nil, err
	}

	// also validate if there is user session present with access token
	sessions := sessionstore.GetUserSessions(claims["id"].(string))
	if len(sessions) == 0 {
		return nil, errors.New("unauthorized")
	}

	return claims, nil
}
