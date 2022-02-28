package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthorizeHandler is the handler for the /authorize route
// required params
// ?redirect_uri = redirect url
// state[recommended] = to prevent CSRF attack (for authorizer its compulsory)
// code_challenge = to prevent CSRF attack
// code_challenge_method = to prevent CSRF attack [only sh256 is supported]
func AuthorizeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		redirectURI := strings.TrimSpace(c.Query("redirect_uri"))
		responseType := strings.TrimSpace(c.Query("response_type"))
		state := strings.TrimSpace(c.Query("state"))
		codeChallenge := strings.TrimSpace(c.Query("code_challenge"))
		codeChallengeMethod := strings.TrimSpace(c.Query("code_challenge_method"))
		fmt.Println(codeChallengeMethod)
		template := "authorize.tmpl"

		if redirectURI == "" {
			c.HTML(http.StatusBadRequest, template, gin.H{
				"targetOrigin":          nil,
				"authorizationResponse": nil,
				"error":                 "redirect_uri is required",
			})
			return
		}

		if state == "" {
			c.HTML(http.StatusBadRequest, template, gin.H{
				"targetOrigin":          nil,
				"authorizationResponse": nil,
				"error":                 "state is required",
			})
			return
		}

		if responseType == "" {
			responseType = "code"
		}

		isCode := responseType == "code"
		isToken := responseType == "token"

		if !isCode && !isToken {
			c.HTML(http.StatusBadRequest, template, gin.H{
				"targetOrigin":          nil,
				"authorizationResponse": nil,
				"error":                 "response_type is invalid",
			})
			return
		}

		if isCode {
			if codeChallenge == "" {
				c.HTML(http.StatusBadRequest, template, gin.H{
					"targetOrigin":          nil,
					"authorizationResponse": nil,
					"error":                 "code_challenge is required",
				})
				return
			}
		}
	}
}
