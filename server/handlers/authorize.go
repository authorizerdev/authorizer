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

const (
	authorizeWebMessageTemplate = "authorize_web_message.tmpl"
	authorizeFormPostTemplate   = "authorize_form_post.tmpl"
)

func AuthorizeHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		redirectURI := strings.TrimSpace(gc.Query("redirect_uri"))
		responseType := strings.TrimSpace(gc.Query("response_type"))
		state := strings.TrimSpace(gc.Query("state"))
		codeChallenge := strings.TrimSpace(gc.Query("code_challenge"))
		scopeString := strings.TrimSpace(gc.Query("scope"))
		clientID := strings.TrimSpace(gc.Query("client_id"))
		responseMode := strings.TrimSpace(gc.Query("response_mode"))
		nonce := strings.TrimSpace(gc.Query("nonce"))

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
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log := log.WithFields(log.Fields{
			"response_mode":  responseMode,
			"response_type":  responseType,
			"state":          state,
			"code_challenge": codeChallenge,
			"scope":          scope,
			"redirect_uri":   redirectURI,
		})

		code := uuid.New().String()
		if nonce == "" {
			nonce = uuid.New().String()
		}
		memorystore.Provider.SetState(codeChallenge, code)

		// used for response mode query or fragment
		loginState := "state=" + state + "&scope=" + strings.Join(scope, " ") + "&redirect_uri=" + redirectURI + "&code=" + code + "&nonce=" + nonce
		loginURL := "/app?" + loginState

		if responseMode == constants.ResponseModeFragment {
			loginURL = "/app#" + loginState
		}

		if state == "" {
			handleResponse(gc, responseMode, loginURL, redirectURI, map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"error":             "state_required",
					"error_description": "state is required",
				},
			}, http.StatusOK)
			return
		}

		if responseType == constants.ResponseTypeCode && codeChallenge == "" {
			handleResponse(gc, responseMode, loginURL, redirectURI, map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"error":             "code_challenge_required",
					"error_description": "code challenge is required",
				},
			}, http.StatusOK)
		}

		loginError := map[string]interface{}{
			"type": "authorization_response",
			"response": map[string]interface{}{
				"error":             "login_required",
				"error_description": "Login is required",
			},
		}
		sessionToken, err := cookie.GetSession(gc)
		if err != nil {
			log.Debug("GetSession failed: ", err)
			handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
			return
		}

		// get session from cookie
		claims, err := token.ValidateBrowserSession(gc, sessionToken)
		if err != nil {
			log.Debug("ValidateBrowserSession failed: ", err)
			handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
			return
		}

		userID := claims.Subject
		user, err := db.Provider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug("GetUserByID failed: ", err)
			handleResponse(gc, responseMode, loginURL, redirectURI, map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"error":             "signup_required",
					"error_description": "Sign up required",
				},
			}, http.StatusOK)
			return
		}

		sessionKey := user.ID
		if claims.LoginMethod != "" {
			sessionKey = claims.LoginMethod + ":" + user.ID
		}

		newSessionTokenData, newSessionToken, err := token.CreateSessionToken(user, nonce, claims.Roles, scope, claims.LoginMethod)
		if err != nil {
			log.Debug("CreateSessionToken failed: ", err)
			handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
			return
		}

		if err := memorystore.Provider.SetState(codeChallenge, code+"@"+newSessionToken); err != nil {
			log.Debug("SetState failed: ", err)
			handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
			return
		}

		// rollover the session for security
		go memorystore.Provider.DeleteUserSession(sessionKey, claims.Nonce)
		if responseType == constants.ResponseTypeCode {
			if err := memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+newSessionTokenData.Nonce, newSessionToken); err != nil {
				log.Debug("SetUserSession failed: ", err)
				handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
				return
			}

			cookie.SetSession(gc, newSessionToken)

			// in case, response type is code and user is already logged in send the code and state
			// and cookie session will already be rolled over and set
			// gc.HTML(http.StatusOK, authorizeWebMessageTemplate, gin.H{
			// 	"target_origin": redirectURI,
			// 	"authorization_response": map[string]interface{}{
			// 		"type": "authorization_response",
			// 		"response": map[string]string{
			// 			"code":  code,
			// 			"state": state,
			// 		},
			// 	},
			// })

			params := "code=" + code + "&state=" + state + "&nonce=" + nonce
			if responseMode == constants.ResponseModeQuery {
				if strings.Contains(redirectURI, "?") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "?" + params
				}
			} else if responseMode == constants.ResponseModeFragment {
				if strings.Contains(redirectURI, "#") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "#" + params
				}
			}

			handleResponse(gc, responseMode, loginURL, redirectURI, map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"code":  code,
					"state": state,
				},
			}, http.StatusOK)

			return
		}

		if responseType == constants.ResponseTypeToken || responseType == constants.ResponseTypeIDToken {
			// rollover the session for security
			authToken, err := token.CreateAuthToken(gc, user, claims.Roles, scope, claims.LoginMethod)
			if err != nil {
				log.Debug("CreateAuthToken failed: ", err)
				handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash); err != nil {
				log.Debug("SetUserSession failed: ", err)
				handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token); err != nil {
				log.Debug("SetUserSession failed: ", err)
				handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
				return
			}

			cookie.SetSession(gc, authToken.FingerPrintHash)

			expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
			if expiresIn <= 0 {
				expiresIn = 1
			}

			// used of query mode
			params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + state + "&id_token=" + authToken.IDToken.Token + "&code=" + code + "&nonce=" + nonce

			res := map[string]interface{}{
				"access_token": authToken.AccessToken.Token,
				"id_token":     authToken.IDToken.Token,
				"state":        state,
				"scope":        scope,
				"token_type":   "Bearer",
				"expires_in":   expiresIn,
				"code":         code,
				"nonce":        nonce,
			}

			if authToken.RefreshToken != nil {
				res["refresh_token"] = authToken.RefreshToken.Token
				params += "&refresh_token=" + authToken.RefreshToken.Token
				memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token)
			}

			if responseMode == constants.ResponseModeQuery {
				if strings.Contains(redirectURI, "?") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "?" + params
				}
			} else if responseMode == constants.ResponseModeFragment {
				if strings.Contains(redirectURI, "#") {
					redirectURI = redirectURI + "&" + params
				} else {
					redirectURI = redirectURI + "#" + params
				}
			}

			handleResponse(gc, responseMode, loginURL, redirectURI, map[string]interface{}{
				"type":     "authorization_response",
				"response": res,
			}, http.StatusOK)
			return
		}

		handleResponse(gc, responseMode, loginURL, redirectURI, loginError, http.StatusOK)
	}
}

func validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge string) error {
	if responseType != constants.ResponseTypeCode && responseType != constants.ResponseTypeToken && responseType != constants.ResponseTypeIDToken {
		return fmt.Errorf("invalid response type %s. 'code' & 'token' are valid response_type", responseMode)
	}

	if responseMode != constants.ResponseModeQuery && responseMode != constants.ResponseModeWebMessage && responseMode != constants.ResponseModeFragment && responseMode != constants.ResponseModeFormPost {
		return fmt.Errorf("invalid response mode %s. 'query', 'fragment', 'form_post' and 'web_message' are valid response_mode", responseMode)
	}

	if client, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID); client != clientID || err != nil {
		return fmt.Errorf("invalid client_id %s", clientID)
	}

	return nil
}

func handleResponse(gc *gin.Context, responseMode, loginURI, redirectURI string, data map[string]interface{}, httpStatusCode int) {
	isAuthenticationRequired := false
	if _, ok := data["response"].(map[string]interface{})["error"]; ok {
		isAuthenticationRequired = true
	}

	switch responseMode {
	case constants.ResponseModeQuery, constants.ResponseModeFragment:
		if isAuthenticationRequired {
			gc.Redirect(http.StatusFound, loginURI)
		} else {
			gc.Redirect(http.StatusFound, redirectURI)
		}
		return
	case constants.ResponseModeWebMessage:
		gc.HTML(httpStatusCode, authorizeWebMessageTemplate, gin.H{
			"target_origin":          redirectURI,
			"authorization_response": data,
		})
		return
	case constants.ResponseModeFormPost:
		gc.HTML(httpStatusCode, authorizeFormPostTemplate, gin.H{
			"target_origin":          redirectURI,
			"authorization_response": data["response"],
		})
		return
	}
}
