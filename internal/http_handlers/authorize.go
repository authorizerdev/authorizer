package http_handlers

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

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/token"
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
func (h *httpProvider) AuthorizeHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "AuthorizeHandler").Logger()
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
			responseMode = h.Config.DefaultAuthorizeResponseMode
		}

		if redirectURI == "" {
			redirectURI = "/app"
		}

		if responseType == "" {
			responseType = h.Config.DefaultAuthorizeResponseType
		}

		if err := h.validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge); err != nil {
			log.Debug().Err(err).Msg("Invalid request")
			gc.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		code := uuid.New().String()
		if nonce == "" {
			nonce = uuid.New().String()
		}

		log = log.With().Str("response_type", responseType).Str("response_mode", responseMode).Str("state", state).Str("code_challenge", codeChallenge).Str("scope", scopeString).Str("client_id", clientID).Str("nonce", nonce).Logger()

		// TODO add state with timeout
		// used for response mode query or fragment
		authState := "state=" + state + "&scope=" + scopeString + "&redirect_uri=" + redirectURI
		if responseType == constants.ResponseTypeCode {
			authState += "&code=" + code
			if err := h.MemoryStoreProvider.SetState(state, code+"@@"+codeChallenge); err != nil {
				log.Debug().Err(err).Msg("Error setting temp code")
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		} else {
			authState += "&nonce=" + nonce
			if err := h.MemoryStoreProvider.SetState(state, nonce); err != nil {
				log.Debug().Err(err).Msg("Error setting temp nonce")
				gc.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
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
			log.Debug().Err(err).Msg("Error getting session token")
			handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
			return
		}

		// get session from cookie
		claims, err := h.TokenProvider.ValidateBrowserSession(gc, sessionToken)
		if err != nil {
			log.Debug().Err(err).Msg("Error validating session token")
			handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
			return
		}

		userID := claims.Subject
		user, err := h.StorageProvider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug().Err(err).Msg("Error getting user")
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
		go h.MemoryStoreProvider.DeleteUserSession(sessionKey, claims.Nonce)
		if responseType == constants.ResponseTypeCode {
			newSessionTokenData, newSessionToken, newSessionExpiresAt, err := h.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
				User:        user,
				Nonce:       nonce,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating session token")
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
			if err := h.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+newSessionToken); err != nil {
				log.Debug().Err(err).Msg("Error setting temp code")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+newSessionTokenData.Nonce, newSessionToken, newSessionExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error setting session token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			cookie.SetSession(gc, newSessionToken, h.Config.AppCookieSecure)

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
			hostname := parsers.GetHost(gc)
			// rollover the session for security
			authToken, err := h.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
				User:        user,
				Nonce:       nonce,
				Roles:       claims.Roles,
				Scope:       scope,
				LoginMethod: claims.LoginMethod,
				HostName:    hostname,
			})
			if err != nil {
				log.Debug().Err(err).Msg("Error creating auth token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+nonce, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error setting session token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt); err != nil {
				log.Debug().Err(err).Msg("Error setting access token")
				handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
				return
			}

			cookie.SetSession(gc, authToken.FingerPrintHash, h.Config.AppCookieSecure)

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
				if err := h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt); err != nil {
					log.Debug().Err(err).Msg("Error setting refresh token")
					handleResponse(gc, responseMode, authURL, redirectURI, loginError, http.StatusOK)
					return
				}
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

func (h *httpProvider) validateAuthorizeRequest(responseType, responseMode, clientID, state, codeChallenge string) error {
	if strings.TrimSpace(state) == "" {
		return fmt.Errorf("invalid state. state is required to prevent csrf attack")
	}
	if responseType != constants.ResponseTypeCode && responseType != constants.ResponseTypeToken && responseType != constants.ResponseTypeIDToken {
		return fmt.Errorf("invalid response type %s. 'code' & 'token' are valid response_type", responseMode)
	}

	if responseMode != constants.ResponseModeQuery && responseMode != constants.ResponseModeWebMessage && responseMode != constants.ResponseModeFragment && responseMode != constants.ResponseModeFormPost {
		return fmt.Errorf("invalid response mode %s. 'query', 'fragment', 'form_post' and 'web_message' are valid response_mode", responseMode)
	}

	if h.Config.ClientID != clientID {
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
