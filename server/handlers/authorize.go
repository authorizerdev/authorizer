package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
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
			responseMode = constants.ResponseModeQuery
		}

		if redirectURI == "" {
			redirectURI = "/app"
		}

		if responseType == "" {
			responseType = "token"
		}

		if err := validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge); err != nil {
			log.Debug("invalid authorization request: ", err)
			gc.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		}

		sessionToken, err := cookie.GetSession(gc)
		if err != nil {
			log.Debug("GetSession failed: ", err)
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("login required. %v", err)})
			return
		}

		// get session from cookie
		claims, err := token.ValidateBrowserSession(gc, sessionToken)
		if err != nil {
			log.Debug("ValidateBrowserSession failed: ", err)
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("login required. %v", err)})
			return
		}
		userID := claims.Subject
		user, err := db.Provider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug("GetUserByID failed: ", err)
			gc.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("sign up required. %v", err)})
			return
		}

		sessionKey := user.ID
		if claims.LoginMethod != "" {
			sessionKey = claims.LoginMethod + ":" + user.ID
		}

		loginState := "state=" + state + "&scope=" + strings.Join(scope, " ") + "&redirect_uri=" + redirectURI
		loginURL := "/app?" + loginState
		if responseMode == constants.ResponseModeFragment {
			loginURL = "/app#" + loginState
		}

		// if user is logged in
		// based on the response type code, generate the response
		if isResponseTypeCode {
			// rollover the session for security
			go memorystore.Provider.DeleteUserSession(sessionKey, claims.Nonce)
			nonce := uuid.New().String()
			newSessionTokenData, newSessionToken, err := token.CreateSessionToken(user, nonce, claims.Roles, scope, claims.LoginMethod)
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

			memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+newSessionTokenData.Nonce, newSessionToken)
			cookie.SetSession(gc, newSessionToken)
			code := uuid.New().String()
			memorystore.Provider.SetState(codeChallenge, code+"@"+newSessionToken)
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
			authToken, err := token.CreateAuthToken(gc, user, claims.Roles, scope, claims.LoginMethod)
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

			go memorystore.Provider.DeleteUserSession(sessionKey, claims.Nonce)
			memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash)
			memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token)
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
				memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token)
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

func validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge string) error {
	if responseType != constants.ResponseTypeCode && responseType != constants.ResponseTypeToken {
		return fmt.Errorf("invalid response type %s. 'code' & 'token' are valid response_type", responseMode)
	}

	if responseMode != constants.ResponseModeQuery && responseMode != constants.ResponseModeWebMessage && responseMode != constants.ResponseModeFragment && responseMode != constants.ResponseModeFormPost {
		return fmt.Errorf("invalid response mode %s. 'query', 'fragment', 'form_post' and 'web_message' are valid response_mode")
	}

	if responseType == constants.ResponseTypeCode && strings.TrimSpace(codeChallenge) == "" {
		return fmt.Errorf("code_challenge is required for %s '%s'", responseType, constants.ResponseTypeCode)
	}

	if client, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID); client != clientID || err != nil {
		return fmt.Errorf("invalid client_id %s", clientID)
	}

	if strings.TrimSpace(state) == "" {
		return fmt.Errorf("state is required")
	}

	return nil
}
