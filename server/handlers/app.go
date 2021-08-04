package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
)

type State struct {
	AuthorizerURL string `json:"authorizerURL"`
	RedirectURL   string `json:"redirectURL"`
}

func AppHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := "http://" + c.Request.Host
		state := c.Query("state")
		var stateObj State

		if state == "" {
			cookie, err := utils.GetAuthToken(c)
			log.Println(`cookie`, cookie)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid state"})
				return
			}

			stateObj.AuthorizerURL = host
			stateObj.RedirectURL = host + "/app"

		} else {
			decodedState, err := base64.StdEncoding.DecodeString(state)
			if err != nil {
				c.JSON(400, gin.H{"error": "[unable to decode state] invalid state"})
				return
			}

			err = json.Unmarshal(decodedState, &stateObj)
			if err != nil {
				c.JSON(400, gin.H{"error": "[unable to parse state] invalid state"})
				return
			}

			// validate redirect url with allowed origins
			if !utils.IsValidRedirectURL(stateObj.RedirectURL) {
				c.JSON(400, gin.H{"error": "invalid redirect url"})
				return
			}

			if stateObj.AuthorizerURL == "" {
				c.JSON(400, gin.H{"error": "invalid authorizer url"})
				return
			}

			// validate host and domain of authorizer url
			if utils.GetDomainName(stateObj.AuthorizerURL) != utils.GetDomainName(host) {
				c.JSON(400, gin.H{"error": "invalid host url"})
				return
			}
		}

		log.Println(gin.H{
			"data": map[string]string{
				"authorizerURL": "http://" + stateObj.AuthorizerURL,
				"redirectURL":   stateObj.RedirectURL,
			},
		})

		// debug the request state
		if pusher := c.Writer.Pusher(); pusher != nil {
			// use pusher.Push() to do server push
			if err := pusher.Push("/app/build/bundle.js", nil); err != nil {
				log.Printf("Failed to push: %v", err)
			}
		}
		c.HTML(http.StatusOK, "app.tmpl", gin.H{
			"data": map[string]string{
				"authorizerURL": stateObj.AuthorizerURL,
				"redirectURL":   stateObj.RedirectURL,
			},
		})
	}
}
