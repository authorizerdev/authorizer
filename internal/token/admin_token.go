package token

import (
	"fmt"

	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// TODO remove if not used
// // CreateAdminAuthToken creates the admin token based on secret key
// func CreateAdminAuthToken(tokenType string, c *gin.Context) (string, error) {
// 	adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
// 	if err != nil {
// 		return "", err
// 	}
// 	return crypto.EncryptPassword(adminSecret)
// }

// GetAdminAuthToken helps in getting the admin token from the request cookie
func (p *provider) GetAdminAuthToken(gc *gin.Context) (string, error) {
	token, err := cookie.GetAdminCookie(gc)
	if err != nil || token == "" {
		return "", fmt.Errorf("unauthorized")
	}
	err = bcrypt.CompareHashAndPassword([]byte(token), []byte(p.config.AdminSecret))
	if err != nil {
		return "", fmt.Errorf(`unauthorized`)
	}

	return token, nil
}

// IsSuperAdmin checks if user is super admin
func (p *provider) IsSuperAdmin(gc *gin.Context) bool {
	token, err := p.GetAdminAuthToken(gc)
	if err != nil {
		if p.config.DisableAdminHeaderAuth {
			return false
		}

		secret := gc.Request.Header.Get("x-authorizer-admin-secret")
		if secret == "" {
			return false
		}
		return secret == p.config.AdminSecret
	}

	return token != ""
}
