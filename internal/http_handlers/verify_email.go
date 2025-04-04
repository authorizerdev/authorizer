package http_handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// VerifyEmailHandler handles the verify email route.
// It verifies email based on JWT token in query string
func (h *httpProvider) VerifyEmailHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "VerifyEmailHandler").Logger()
	return func(c *gin.Context) {
		redirectURL := strings.TrimSpace(c.Query("redirect_uri"))
		errorRes := gin.H{
			"error": "token is required",
		}
		tokenInQuery := c.Query("token")
		if tokenInQuery == "" {
			log.Debug().Msg("Token is missing")
			utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}

		verificationRequest, err := h.StorageProvider.GetVerificationRequestByToken(c, tokenInQuery)
		if err != nil {
			log.Debug().Err(err).Msg("Error getting verification request")
			errorRes["error"] = err.Error()
			utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}

		// verify if token exists in db
		hostname := parsers.GetHost(c)
		claim, err := h.TokenProvider.ParseJWTToken(tokenInQuery)
		if err != nil {
			log.Debug().Err(err).Msg("Error parsing jwt token")
			errorRes["error"] = err.Error()
			utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}
		// // hostname, verificationRequest.Nonce, verificationRequest.Email
		if ok, err := h.TokenProvider.ValidateJWTClaims(claim, &token.AuthTokenConfig{
			HostName: hostname,
			Nonce:    verificationRequest.Nonce,
			User: &schemas.User{
				Email: refs.NewStringRef(verificationRequest.Email),
			},
		}); !ok || err != nil {
			log.Debug().Err(err).Msg("Error validating jwt token")
			errorRes["error"] = err.Error()
			utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}

		user, err := h.StorageProvider.GetUserByEmail(c, verificationRequest.Email)
		if err != nil {
			log.Debug().Err(err).Msg("Error getting user by email")
			errorRes["error"] = err.Error()
			utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}

		isSignUp := false
		// update email_verified_at in users table
		if user.EmailVerifiedAt == nil {
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
			isSignUp = true
			user, err = h.StorageProvider.UpdateUser(c, user)
			if err != nil {
				log.Debug().Err(err).Msg("Error updating user")
				errorRes["error"] = err.Error()
				utils.HandleRedirectORJsonResponse(c, http.StatusBadRequest, errorRes, generateRedirectURL(redirectURL, errorRes))
				return
			}
		}
		// delete from verification table
		if err := h.StorageProvider.DeleteVerificationRequest(c, verificationRequest); err != nil {
			log.Debug().Err(err).Msg("Error deleting verification request")
		}

		state := strings.TrimSpace(c.Query("state"))
		rolesString := strings.TrimSpace(c.Query("roles"))
		var roles []string
		if rolesString == "" {
			roles = strings.Split(user.Roles, ",")
		} else {
			roles = strings.Split(rolesString, ",")
		}

		scopeString := strings.TrimSpace(c.Query("scope"))
		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "email", "profile"}
		} else {
			scope = strings.Split(scopeString, " ")
		}
		loginMethod := constants.AuthRecipeMethodBasicAuth
		if verificationRequest.Identifier == constants.VerificationTypeMagicLinkLogin {
			loginMethod = constants.AuthRecipeMethodMagicLinkLogin
		}

		code := ""
		// Not required as /oauth/token cannot be resumed from other tab
		// codeChallenge := ""
		nonce := ""
		if state != "" {
			// Get state from store
			authorizeState, _ := h.MemoryStoreProvider.GetState(state)
			if authorizeState != "" {
				authorizeStateSplit := strings.Split(authorizeState, "@@")
				if len(authorizeStateSplit) > 1 {
					code = authorizeStateSplit[0]
					// Not required as /oauth/token cannot be resumed from other tab
					// codeChallenge = authorizeStateSplit[1]
				} else {
					nonce = authorizeState
				}
				go h.MemoryStoreProvider.RemoveState(state)
			}
		}
		if nonce == "" {
			nonce = uuid.New().String()
		}
		authToken, err := h.TokenProvider.CreateAuthToken(c, &token.AuthTokenConfig{
			User:        user,
			Roles:       roles,
			Scope:       scope,
			LoginMethod: loginMethod,
			Nonce:       nonce,
			Code:        code,
			HostName:    hostname,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Error creating auth token")
			errorRes["error"] = err.Error()
			utils.HandleRedirectORJsonResponse(c, http.StatusInternalServerError, errorRes, generateRedirectURL(redirectURL, errorRes))
			return
		}

		// Code challenge could be optional if PKCE flow is not used
		// Not required as /oauth/token cannot be resumed from other tab
		// if code != "" {
		// 	if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
		// 		log.Debug("Error setting code state ", err)
		// 		errorRes["error"] = err.Error()
		// 		c.JSON(500, errorRes)
		// 		return
		// 	}
		// }

		expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		if expiresIn <= 0 {
			expiresIn = 1
		}

		params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + state + "&id_token=" + authToken.IDToken.Token + "&nonce=" + nonce

		if code != "" {
			params += "&code=" + code
		}

		sessionKey := loginMethod + ":" + user.ID
		cookie.SetSession(c, authToken.FingerPrintHash)
		h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
		h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

		if authToken.RefreshToken != nil {
			params = params + `&refresh_token=` + authToken.RefreshToken.Token
			h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
		}

		if redirectURL == "" {
			redirectURL = claim["redirect_uri"].(string)
		}

		if strings.Contains(redirectURL, "?") {
			redirectURL = redirectURL + "&" + params
		} else {
			redirectURL = redirectURL + "?" + strings.TrimPrefix(params, "&")
		}

		go func() {
			if isSignUp {
				h.EventsProvider.RegisterEvent(c, constants.UserSignUpWebhookEvent, loginMethod, user)
				// User is also logged in with signup
				h.EventsProvider.RegisterEvent(c, constants.UserLoginWebhookEvent, loginMethod, user)
			} else {
				h.EventsProvider.RegisterEvent(c, constants.UserLoginWebhookEvent, loginMethod, user)
			}
			if err := h.StorageProvider.AddSession(c, &schemas.Session{
				UserID:    user.ID,
				UserAgent: utils.GetUserAgent(c.Request),
				IP:        utils.GetIP(c.Request),
			}); err != nil {
				log.Debug().Err(err).Msg("Error adding session")
			}
		}()

		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

func generateRedirectURL(url string, res map[string]interface{}) string {
	redirectURL := url
	if redirectURL == "" {
		return ""
	}
	var paramsArr []string
	for key, value := range res {
		paramsArr = append(paramsArr, key+"="+value.(string))
	}
	params := strings.Join(paramsArr, "&")
	if strings.Contains(redirectURL, "?") {
		redirectURL = redirectURL + "&" + params
	} else {
		redirectURL = redirectURL + "?" + strings.TrimPrefix(params, "&")
	}
	return redirectURL
}
