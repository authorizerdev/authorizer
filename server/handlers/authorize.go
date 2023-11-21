package handlers

/**
LOGIC TO REMEMBER THE AUTHORIZE FLOW


jargons
`at_hash` -> access_token_hash
`c_hash` -> code_hash


# ResponseType: Code
	with /authorize request
		- set state [state, code@@challenge]
		- add &code to login redirect url
	login resolver has optional param state
		-if state found in store, split with @@
		- if len > 1 -> response type is code and has code + challenge
		- set `nonce, code` for createAuthToken request so that `c_hash` can be generated
		- do not add `nonce` to id_token in code flow, instead set `c_hash` and `at_hash`


# ResponseType: token / id_token
	with /authorize request
		- set state [state, nonce]
		- add &nonce to login redirect url
	login resolver has optional param state
		- if state found in store, split with @@
		- if len < 1 -> response type is token / id_token and value is nonce
		- send received nonce for createAuthToken with empty code value
		- set `nonce` and `at_hash` in `id_token`
**/

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
)

// Check the flow for generating and verifying codes: https://developer.okta.com/blog/2019/08/22/okta-authjs-pkce#:~:text=PKCE%20works%20by%20having%20the,is%20called%20the%20Code%20Challenge.

// Check following docs for understanding request / response params for various types of requests: https://auth0.com/docs/authenticate/login/oidc-conformant-authentication/oidc-adoption-auth-code-flow

const (
	authorizeWebMessageTemplate = "authorize_web_message.tmpl"
	authorizeFormPostTemplate   = "authorize_form_post.tmpl"
	baseAppPath                 = "/app"
	signupPath                  = "/app/signup"
)

// AuthorizeHandler is the handler for the /authorize route
// required params
// ?redirect_uri = redirect url
// ?response_mode = to decide if result should be html or re-direct
// state[recommended] = to prevent CSRF attack (for authorizer its compulsory)
// code_challenge = to prevent CSRF attack
// code_challenge_method = to prevent CSRF attack [only sh256 is supported]
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
		screenHint := strings.TrimSpace(gc.Query("screen_hint"))

		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "profile", "email"}
		} else {
			scope = strings.Split(scopeString, " ")
		}

		if responseMode == "" {
			if val, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultAuthorizeResponseMode); err == nil {
				responseMode = val
			} else {
				responseMode = constants.ResponseModeQuery
			}
		}

		if redirectURI == "" {
			redirectURI = "/app"
		}

		if responseType == "" {
			if val, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultAuthorizeResponseType); err == nil {
				responseType = val
			} else {
				responseType = constants.ResponseTypeToken
			}
		}

		if err := validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge); err != nil {
			log.Debug("invalid authorization request: ", err)
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		code := uuid.New().String()
		if nonce == "" {
			nonce = uuid.New().String()
		}

		log := log.WithFields(log.Fields{
			"response_mode": responseMode,
			"response_type": responseType,
		})

		// TODO add state with timeout
		// used for response mode query or fragment
		authState := "state=" + state + "&scope=" + strings.Join(scope, " ") + "&redirect_uri=" + redirectURI
		if responseType == constants.ResponseTypeCode {
			authState += "&code=" + code
			if err := memorystore.Provider.SetState(state, code+"@@"+codeChallenge); err != nil {
				log.Debug("Error setting temp code", err)
			}
		} else {
			authState += "&nonce=" + nonce
			if err := memorystore.Provider.SetState(state, nonce); err != nil {
				log.Debug("Error setting temp code", err)
			}
		}

		authURL := baseAppPath + "?" + authState

		if screenHint == constants.ScreenHintSignUp {
			authURL = signupPath + "?" + authState
		}

		if responseMode == constants.ResponseModeFragment && screenHint == constants.ScreenHintSignUp {
			authURL = signupPath + "#" + authState
		} else if responseMode == constants.ResponseModeFragment {
			authURL = baseAppPath + "#" + authState
		}

		if responseType == constants.ResponseTypeCode && codeChallenge == "" {
			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
				"type": "authorization_response",
				"response": map[string]interface{}{
					"error":             "code_challenge_required",
					"error_description": "code challenge is required",
				},
			}, http.StatusOK)
			return
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
			handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
			return
		}

		// get session from cookie
		claims, err := token.ValidateBrowserSession(gc, sessionToken)
		if err != nil {
			log.Debug("ValidateBrowserSession failed: ", err)
			handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
			return
		}

		userID := claims.Subject
		user, err := db.Provider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug("GetUserByID failed: ", err)
			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
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

		// rollover the session for security
		go memorystore.Provider.DeleteUserSession(sessionKey, claims.Nonce)
		if responseType == constants.ResponseTypeCode {
			newSessionTokenData, newSessionToken, newSessionExpiresAt, err := token.CreateSessionToken(user, nonce, claims.Roles, scope, claims.LoginMethod)
			if err != nil {
				log.Debug("CreateSessionToken failed: ", err)
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			// TODO: add state with timeout
			// if err := memorystore.Provider.SetState(codeChallenge, code+"@"+newSessionToken); err != nil {
			// 	log.Debug("SetState failed: ", err)
			// 	handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
			// 	return
			// }

			// TODO: add state with timeout
			if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+newSessionToken); err != nil {
				log.Debug("SetState failed: ", err)
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+newSessionTokenData.Nonce, newSessionToken, newSessionExpiresAt); err != nil {
				log.Debug("SetUserSession failed: ", err)
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
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

			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
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
			authToken, err := token.CreateAuthToken(gc, user, claims.Roles, scope, claims.LoginMethod, nonce, "")
			if err != nil {
				log.Debug("CreateAuthToken failed: ", err)
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+nonce, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt); err != nil {
				log.Debug("SetUserSession failed: ", err)
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt); err != nil {
				log.Debug("SetUserSession failed: ", err)
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			cookie.SetSession(gc, authToken.FingerPrintHash)

			// used of query mode
			params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(authToken.IDToken.ExpiresAt, 10) + "&state=" + state + "&id_token=" + authToken.IDToken.Token

			res := map[string]interface{}{
				"access_token": authToken.AccessToken.Token,
				"id_token":     authToken.IDToken.Token,
				"state":        state,
				"scope":        strings.Join(scope, " "),
				"token_type":   "Bearer",
				"expires_in":   authToken.AccessToken.ExpiresAt,
			}

			if nonce != "" {
				params += "&nonce=" + nonce
				res["nonce"] = nonce
			}

			if authToken.RefreshToken != nil {
				res["refresh_token"] = authToken.RefreshToken.Token
				params += "&refresh_token=" + authToken.RefreshToken.Token
				memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
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

			handleResponse(gc, responseMode, authURL, redirectURI, map[string]interface{}{
				"type":     "authorization_response",
				"response": res,
			}, http.StatusOK)
			return
		}

		handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
	}
}

func validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge string) error {
	if strings.TrimSpace(state) == "" {
		return fmt.Errorf("invalid state. state is required to prevent csrf attack")
	}
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

func handleResponse(gc *gin.Context, responseMode, authURI, redirectURI string, data map[string]interface{}, httpStatusCode int) {
	isAuthenticationRequired := false
	if _, ok := data["response"].(map[string]interface{})["error"]; ok {
		isAuthenticationRequired = true
	}

	if isAuthenticationRequired && responseMode != constants.ResponseModeWebMessage {
		gc.Redirect(http.StatusFound, authURI)
		return
	}

	switch responseMode {
	case constants.ResponseModeQuery, constants.ResponseModeFragment:
		gc.Redirect(http.StatusFound, redirectURI)
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
