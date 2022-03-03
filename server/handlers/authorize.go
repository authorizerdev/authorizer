package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthorizeHandler is the handler for the /authorize route
// required params
// ?redirect_uri = redirect url
// state[recommended] = to prevent CSRF attack (for authorizer its compulsory)
// code_challenge = to prevent CSRF attack
// code_challenge_method = to prevent CSRF attack [only sh256 is supported]

// check the flow for generating and verifying codes: https://developer.okta.com/blog/2019/08/22/okta-authjs-pkce#:~:text=PKCE%20works%20by%20having%20the,is%20called%20the%20Code%20Challenge.
func AuthorizeHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		redirectURI := strings.TrimSpace(gc.Query("redirect_uri"))
		responseType := strings.TrimSpace(gc.Query("response_type"))
		state := strings.TrimSpace(gc.Query("state"))
		codeChallenge := strings.TrimSpace(gc.Query("code_challenge"))
		template := "authorize.tmpl"

		if redirectURI == "" {
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": nil,
				"authorization_response": map[string]interface{}{
					"type": "authorization_response",
					"response": map[string]string{
						"error": "redirect_uri is required",
					},
				},
			})
			return
		}

		if state == "" {
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": nil,
				"authorization_response": map[string]interface{}{
					"type": "authorization_response",
					"response": map[string]string{
						"error": "state is required",
					},
				},
			})
			return
		}

		if responseType == "" {
			responseType = "token"
		}

		isResponseTypeCode := responseType == "code"
		isResponseTypeToken := responseType == "token"

		if !isResponseTypeCode && !isResponseTypeToken {
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": nil,
				"authorization_response": map[string]interface{}{
					"type": "authorization_response",
					"response": map[string]string{
						"error": "response_type is invalid",
					},
				},
			})
			return
		}

		if isResponseTypeCode {
			if codeChallenge == "" {
				gc.HTML(http.StatusBadRequest, template, gin.H{
					"target_origin": nil,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error": "code_challenge is required",
						},
					},
				})
				return
			}
		}

		sessionToken, err := cookie.GetSession(gc)
		if err != nil {
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": nil,
				"authorization_response": map[string]interface{}{
					"type": "authorization_response",
					"response": map[string]string{
						"error":             "login_required",
						"error_description": "Login is required",
					},
				},
			})
			return
		}

		// get session from cookie
		claims, err := token.ValidateBrowserSession(gc, sessionToken)
		if err != nil {
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": nil,
				"authorization_response": map[string]interface{}{
					"type": "authorization_response",
					"response": map[string]string{
						"error":             "login_required",
						"error_description": "Login is required",
					},
				},
			})
			return
		}
		userID := claims.Subject
		user, err := db.Provider.GetUserByID(userID)
		if err != nil {
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": nil,
				"authorization_response": map[string]interface{}{
					"type": "authorization_response",
					"response": map[string]string{
						"error":             "signup_required",
						"error_description": "Sign up required",
					},
				},
			})
			return
		}

		// if user is logged in
		// based on the response type, generate the response
		if isResponseTypeCode {
			// rollover the session for security
			sessionstore.RemoveState(sessionToken)
			nonce := uuid.New().String()
			newSessionTokenData, newSessionToken, err := token.CreateSessionToken(user, nonce, claims.Roles, claims.Scope)
			if err != nil {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": nil,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error":             "login_required",
							"error_description": "Login is required",
						},
					},
				})
				return
			}

			sessionstore.SetState(newSessionToken, newSessionTokenData.Nonce+"@"+user.ID)
			cookie.SetSession(gc, newSessionToken)
			code := uuid.New().String()
			sessionstore.SetState("code_challenge_"+codeChallenge, code)
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": redirectURI,
				"authorization_response": map[string]string{
					"code":  code,
					"state": state,
				},
			})
			return
		}

		if isResponseTypeToken {
			// rollover the session for security
			authToken, err := token.CreateAuthToken(gc, user, claims.Roles, claims.Scope)
			if err != nil {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": nil,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error":             "login_required",
							"error_description": "Login is required",
						},
					},
				})
				return
			}
			sessionstore.RemoveState(sessionToken)
			sessionstore.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
			sessionstore.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)
			cookie.SetSession(gc, authToken.FingerPrintHash)
			expiresIn := int64(1800)
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": redirectURI,
				"authorization_response": map[string]interface{}{
					"access_token": authToken.AccessToken.Token,
					"id_token":     authToken.IDToken.Token,
					"state":        state,
					"scope":        claims.Scope,
					"expires_in":   expiresIn,
				},
			})
			return
		}
		fmt.Println("=> returning from here...")

		// by default return with error
		gc.HTML(http.StatusOK, template, gin.H{
			"target_origin": nil,
			"authorization_response": map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]string{
					"error":             "login_required",
					"error_description": "Login is required",
				},
			},
		})
	}
}
