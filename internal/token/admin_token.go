package token

import (
	"fmt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// CreateAdminAuthToken creates the admin token based on secret key
func CreateAdminAuthToken(tokenType string, c *gin.Context) (string, error) {
	adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
	if err != nil {
		return "", err
	}
	return crypto.EncryptPassword(adminSecret)
}

// GetAdminAuthToken helps in getting the admin token from the request cookie
func GetAdminAuthToken(gc *gin.Context) (string, error) {
	token, err := cookie.GetAdminCookie(gc)
	if err != nil || token == "" {
		return "", fmt.Errorf("unauthorized")
	}

	adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
	if err != nil {
		return "", err
	}
	err = bcrypt.CompareHashAndPassword([]byte(token), []byte(adminSecret))

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
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		if err != nil {
			return false
		}
		return secret == adminSecret
	}

	return token != ""
}
