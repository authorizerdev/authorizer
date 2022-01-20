package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/robertkrimen/otto"
	"golang.org/x/crypto/bcrypt"
)

// CreateAuthToken util to create JWT token, based on
// user information, roles config and CUSTOM_ACCESS_TOKEN_SCRIPT
func CreateAuthToken(user db.User, tokenType string, roles []string) (string, int64, error) {
	t := jwt.New(jwt.GetSigningMethod(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtType)))
	expiryBound := time.Hour
	if tokenType == constants.TokenTypeRefreshToken {
		// expires in 1 year
		expiryBound = time.Hour * 8760
	}

	expiresAt := time.Now().Add(expiryBound).Unix()

	resUser := GetResponseUserData(user)
	userBytes, _ := json.Marshal(&resUser)
	var userMap map[string]interface{}
	json.Unmarshal(userBytes, &userMap)

	claimKey := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtRoleClaim)
	customClaims := jwt.MapClaims{
		"exp":           expiresAt,
		"iat":           time.Now().Unix(),
		"token_type":    tokenType,
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

// GetAuthToken helps in getting the JWT token from the
// request cookie or authorization header
func GetAuthToken(gc *gin.Context) (string, error) {
	token, err := GetCookie(gc)
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

// VerifyAuthToken helps in verifying the JWT token
func VerifyAuthToken(token string) (map[string]interface{}, error) {
	var res map[string]interface{}
	claims := jwt.MapClaims{}

	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyJwtSecret)), nil
	})
	if err != nil {
		return res, err
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

// CreateAdminAuthToken creates the admin token based on secret key
func CreateAdminAuthToken(tokenType string, c *gin.Context) (string, error) {
	return EncryptPassword(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
}

// GetAdminAuthToken helps in getting the admin token from the request cookie
func GetAdminAuthToken(gc *gin.Context) (string, error) {
	token, err := GetAdminCookie(gc)
	if err != nil || token == "" {
		return "", fmt.Errorf("unauthorized")
	}

	// cookie escapes special characters like $
	// hence we need to unescape before comparing
	decodedValue, err := url.QueryUnescape(token)
	if err != nil {
		return "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(decodedValue), []byte(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)))
	log.Println("error comparing hash:", err)
	if err != nil {
		return "", fmt.Errorf(`unauthorized`)
	}

	return token, nil
}
