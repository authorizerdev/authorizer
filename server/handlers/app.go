package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
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
		state := c.Query("state")

		var stateObj State

		if state == "" {
			// cookie, err := utils.GetAuthToken(c)
			// if err != nil {
			// 	c.JSON(400, gin.H{"error": "invalid state"})
			// 	return
			// }

			stateObj.AuthorizerURL = constants.EnvData.AUTHORIZER_URL
			stateObj.RedirectURL = constants.EnvData.AUTHORIZER_URL + "/app"

		} else {
			decodedState, err := utils.DecryptB64(state)
			if err != nil {
				c.JSON(400, gin.H{"error": "[unable to decode state] invalid state"})
				return
			}

			err = json.Unmarshal([]byte(decodedState), &stateObj)
			if err != nil {
				c.JSON(400, gin.H{"error": "[unable to parse state] invalid state"})
				return
			}
			stateObj.AuthorizerURL = strings.TrimSuffix(stateObj.AuthorizerURL, "/")
			stateObj.RedirectURL = strings.TrimSuffix(stateObj.RedirectURL, "/")

			// validate redirect url with allowed origins
			if !utils.IsValidOrigin(stateObj.RedirectURL) {
				c.JSON(400, gin.H{"error": "invalid redirect url"})
				return
			}

			if stateObj.AuthorizerURL == "" {
				c.JSON(400, gin.H{"error": "invalid authorizer url"})
				return
			}

			// validate host and domain of authorizer url
			if strings.TrimSuffix(stateObj.AuthorizerURL, "/") != constants.EnvData.AUTHORIZER_URL {
				c.JSON(400, gin.H{"error": "invalid host url"})
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
			"data": map[string]string{
				"authorizerURL":    stateObj.AuthorizerURL,
				"redirectURL":      stateObj.RedirectURL,
				"organizationName": constants.EnvData.ORGANIZATION_NAME,
				"organizationLogo": constants.EnvData.ORGANIZATION_LOGO,
			},
		})
	}
}
