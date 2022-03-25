package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthorizeHandler is the handler for the /authorize route
// required params
// ?redirect_uri = redirect url
// ?response_mode = to decide if result should be html or re-direct
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
		scopeString := strings.TrimSpace(gc.Query("scope"))
		clientID := strings.TrimSpace(gc.Query("client_id"))
		template := "authorize.tmpl"
		responseMode := strings.TrimSpace(gc.Query("response_mode"))

		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "profile", "email"}
		} else {
			scope = strings.Split(scopeString, " ")
		}

		if responseMode == "" {
			responseMode = "query"
		}

		if responseMode != "query" && responseMode != "web_message" {
			gc.JSON(400, gin.H{"error": "invalid response mode"})
		}

		fmt.Println("=> redirect URI:", redirectURI)
		fmt.Println("=> state:", state)
		if redirectURI == "" {
			redirectURI = "/app"
		}

		isQuery := responseMode == "query"

		loginURL := "/app?state=" + state + "&scope=" + strings.Join(scope, " ") + "&redirect_uri=" + redirectURI

		if clientID == "" {
			if isQuery {
				gc.Redirect(http.StatusFound, loginURL)
			} else {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": redirectURI,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error": "client_id is required",
						},
					},
				})
			}
			return
		}

		if clientID != envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID) {
			if isQuery {
				gc.Redirect(http.StatusFound, loginURL)
			} else {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": redirectURI,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error": "invalid_client_id",
						},
					},
				})
			}
			return
		}

		if state == "" {
			if isQuery {
				gc.Redirect(http.StatusFound, loginURL)
			} else {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": redirectURI,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error": "state is required",
						},
					},
				})
			}
			return
		}

		if responseType == "" {
			responseType = "token"
		}

		isResponseTypeCode := responseType == "code"
		isResponseTypeToken := responseType == "token"

		if !isResponseTypeCode && !isResponseTypeToken {
			if isQuery {
				gc.Redirect(http.StatusFound, loginURL)
			} else {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": redirectURI,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error": "response_type is invalid",
						},
					},
				})
			}
			return
		}

		if isResponseTypeCode {
			if codeChallenge == "" {
				if isQuery {
					gc.Redirect(http.StatusFound, loginURL)
				} else {
					gc.HTML(http.StatusBadRequest, template, gin.H{
						"target_origin": redirectURI,
						"authorization_response": map[string]interface{}{
							"type": "authorization_response",
							"response": map[string]string{
								"error": "code_challenge is required",
							},
						},
					})
				}
				return
			}
		}

		sessionToken, err := cookie.GetSession(gc)
		if err != nil {
			if isQuery {
				gc.Redirect(http.StatusFound, loginURL)
			} else {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": redirectURI,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error":             "login_required",
							"error_description": "Login is required",
						},
					},
				})
			}
			return
		}

		// get session from cookie
		claims, err := token.ValidateBrowserSession(gc, sessionToken)
		if err != nil {
			if isQuery {
				gc.Redirect(http.StatusFound, loginURL)
			} else {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": redirectURI,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error":             "login_required",
							"error_description": "Login is required",
						},
					},
				})
			}
			return
		}
		userID := claims.Subject
		user, err := db.Provider.GetUserByID(userID)
		if err != nil {
			if isQuery {
				gc.Redirect(http.StatusFound, loginURL)
			} else {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": redirectURI,
					"authorization_response": map[string]interface{}{
						"type": "authorization_response",
						"response": map[string]string{
							"error":             "signup_required",
							"error_description": "Sign up required",
						},
					},
				})
			}
			return
		}

		// if user is logged in
		// based on the response type, generate the response
		if isResponseTypeCode {
			// rollover the session for security
			sessionstore.RemoveState(sessionToken)
			nonce := uuid.New().String()
			newSessionTokenData, newSessionToken, err := token.CreateSessionToken(user, nonce, claims.Roles, scope)
			if err != nil {
				if isQuery {
					gc.Redirect(http.StatusFound, loginURL)
				} else {
					gc.HTML(http.StatusOK, template, gin.H{
						"target_origin": redirectURI,
						"authorization_response": map[string]interface{}{
							"type": "authorization_response",
							"response": map[string]string{
								"error":             "login_required",
								"error_description": "Login is required",
							},
						},
					})
				}
				return
			}

			sessionstore.SetState(newSessionToken, newSessionTokenData.Nonce+"@"+user.ID)
			cookie.SetSession(gc, newSessionToken)
			code := uuid.New().String()
			sessionstore.SetState(codeChallenge, code+"@"+newSessionToken)
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": redirectURI,
				"authorization_response": map[string]interface{}{
					"type": "authorization_response",
					"response": map[string]string{
						"code":  code,
						"state": state,
					},
				},
			})
			return
		}

		if isResponseTypeToken {
			// rollover the session for security
			authToken, err := token.CreateAuthToken(gc, user, claims.Roles, scope)
			if err != nil {
				if isQuery {
					gc.Redirect(http.StatusFound, loginURL)
				} else {
					gc.HTML(http.StatusOK, template, gin.H{
						"target_origin": redirectURI,
						"authorization_response": map[string]interface{}{
							"type": "authorization_response",
							"response": map[string]string{
								"error":             "login_required",
								"error_description": "Login is required",
							},
						},
					})
				}
				return
			}
			sessionstore.RemoveState(sessionToken)
			sessionstore.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
			sessionstore.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)
			cookie.SetSession(gc, authToken.FingerPrintHash)

			expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
			if expiresIn <= 0 {
				expiresIn = 1
			}

			// used of query mode
			params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + state + "&id_token=" + authToken.IDToken.Token

			res := map[string]interface{}{
				"access_token": authToken.AccessToken.Token,
				"id_token":     authToken.IDToken.Token,
				"state":        state,
				"scope":        scope,
				"token_type":   "Bearer",
				"expires_in":   expiresIn,
			}

			if authToken.RefreshToken != nil {
				res["refresh_token"] = authToken.RefreshToken.Token
				params += "&refresh_token=" + authToken.RefreshToken.Token
				sessionstore.SetState(authToken.RefreshToken.Token, authToken.FingerPrint+"@"+user.ID)
			}

			if isQuery {
				if strings.Contains(redirectURI, "?") {
					gc.Redirect(http.StatusFound, redirectURI+"&"+params)
				} else {
					gc.Redirect(http.StatusFound, redirectURI+"?"+params)
				}
			} else {
				gc.HTML(http.StatusOK, template, gin.H{
					"target_origin": redirectURI,
					"authorization_response": map[string]interface{}{
						"type":     "authorization_response",
						"response": res,
					},
				})
			}
			return
		}

		if isQuery {
			gc.Redirect(http.StatusFound, loginURL)
		} else {
			// by default return with error
			gc.HTML(http.StatusOK, template, gin.H{
				"target_origin": redirectURI,
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
}
