package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
)

// State is the struct that holds authorizer url and redirect url
// They are provided via query string in the request
type State struct {
	AuthorizerURL string `json:"authorizerURL"`
	RedirectURL   string `json:"redirectURL"`
}

// AppHandler is the handler for the /app route
func AppHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		hostname := utils.GetHost(c)
		if envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableLoginPage) {
			c.JSON(400, gin.H{"error": "login page is not enabled"})
			return
		}

		redirect_uri := strings.TrimSpace(c.Query("redirect_uri"))
		state := strings.TrimSpace(c.Query("state"))
		scopeString := strings.TrimSpace(c.Query("scope"))

		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "profile", "email"}
		} else {
			scope = strings.Split(scopeString, " ")
		}

		if redirect_uri == "" {
			redirect_uri = hostname + "/app"
		} else {
			// validate redirect url with allowed origins
			if !utils.IsValidOrigin(redirect_uri) {
				c.JSON(400, gin.H{"error": "invalid redirect url"})
				return
			}
		}

		// debug the request state
		if pusher := c.Writer.Pusher(); pusher != nil {
			// use pusher.Push() to do server push
			if err := pusher.Push("/app/build/bundle.js", nil); err != nil {
				log.Printf("Failed to push: %v", err)
			}
		}
		c.HTML(http.StatusOK, "app.tmpl", gin.H{
			"data": map[string]interface{}{
				"authorizerURL":    hostname,
				"redirectURL":      redirect_uri,
				"scope":            scope,
				"state":            state,
				"organizationName": envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyOrganizationName),
				"organizationLogo": envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyOrganizationLogo),
			},
		})
	}
}
