package token

import (
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// CreateAdminAuthToken creates the admin token based on secret key
func CreateAdminAuthToken(tokenType string, c *gin.Context) (string, error) {
	return utils.EncryptPassword(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
}

// GetAdminAuthToken helps in getting the admin token from the request cookie
func GetAdminAuthToken(gc *gin.Context) (string, error) {
	token, err := cookie.GetAdminCookie(gc)
	if err != nil || token == "" {
		return "", fmt.Errorf("unauthorized")
	}

	err = bcrypt.CompareHashAndPassword([]byte(token), []byte(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)))

	if err != nil {
		return "", fmt.Errorf(`unauthorized`)
	}

	return token, nil
}

// IsSuperAdmin checks if user is super admin
func IsSuperAdmin(gc *gin.Context) bool {
	token, err := GetAdminAuthToken(gc)
	if err != nil {
		secret := gc.Request.Header.Get("x-authorizer-admin-secret")
		if secret == "" {
			return false
		}

		return secret == envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
	}

	return token != ""
}
