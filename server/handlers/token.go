package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
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

type RequestBody struct {
	CodeVerifier string `form:"code_verifier" json:"code_verifier"`
	Code         string `form:"code" json:"code"`
	ClientID     string `form:"client_id" json:"client_id"`
	ClientSecret string `form:"client_secret" json:"client_secret"`
	GrantType    string `form:"grant_type" json:"grant_type"`
	RefreshToken string `form:"refresh_token" json:"refresh_token"`
	RedirectURI  string `form:"redirect_uri" json:"redirect_uri"`
}

// TokenHandler to handle /oauth/token requests
// grant type required
func TokenHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		var reqBody RequestBody
		if err := gc.Bind(&reqBody); err != nil {
			log.Debug("Error binding JSON: ", err)
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "error_binding_json",
				"error_description": err.Error(),
			})
			return
		}

		codeVerifier := strings.TrimSpace(reqBody.CodeVerifier)
		code := strings.TrimSpace(reqBody.Code)
		clientID := strings.TrimSpace(reqBody.ClientID)
		grantType := strings.TrimSpace(reqBody.GrantType)
		refreshToken := strings.TrimSpace(reqBody.RefreshToken)
		clientSecret := strings.TrimSpace(reqBody.ClientSecret)

		if grantType == "" {
			grantType = "authorization_code"
		}

		isRefreshTokenGrant := grantType == "refresh_token"
		isAuthorizationCodeGrant := grantType == "authorization_code"

		if !isRefreshTokenGrant && !isAuthorizationCodeGrant {
			log.Debug("Invalid grant type: ", grantType)
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_grant_type",
				"error_description": "grant_type is invalid",
			})
		}

		// check if clientID & clientSecret are present as part of
		// authorization header with basic auth
		if clientID == "" && clientSecret == "" {
			clientID, clientSecret, _ = gc.Request.BasicAuth()
		}

		if clientID == "" {
			log.Debug("Client ID is empty")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "client_id_required",
				"error_description": "The client id is required",
			})
			return
		}

		if client, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID); clientID != client || err != nil {
			log.Debug("Client ID is invalid: ", clientID)
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client_id",
				"error_description": "The client id is invalid",
			})
			return
		}

		var userID string
		var roles, scope []string
		loginMethod := ""
		sessionKey := ""

		if isAuthorizationCodeGrant {
			if code == "" {
				log.Debug("Code is empty")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_code",
					"error_description": "The code is required",
				})
				return
			}

			if codeVerifier == "" && clientSecret == "" {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_dat",
					"error_description": "The code verifier or client secret is required",
				})
				return
			}
			// Get state
			sessionData, err := memorystore.Provider.GetState(code)
			if sessionData == "" || err != nil {
				log.Debug("Session data is empty")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_code",
					"error_description": "The code is invalid",
				})
				return
			}

			// [0] -> code_challenge
			// [1] -> session cookie
			sessionDataSplit := strings.Split(sessionData, "@@")

			go memorystore.Provider.RemoveState(code)

			if codeVerifier != "" {
				hash := sha256.New()
				hash.Write([]byte(codeVerifier))
				encryptedCode := strings.ReplaceAll(base64.RawURLEncoding.EncodeToString(hash.Sum(nil)), "+", "-")
				encryptedCode = strings.ReplaceAll(encryptedCode, "/", "_")
				encryptedCode = strings.ReplaceAll(encryptedCode, "=", "")
				if encryptedCode != sessionDataSplit[0] {
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_code_verifier",
						"error_description": "The code verifier is invalid",
					})
					return
				}

			} else {
				if clientHash, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientSecret); clientSecret != clientHash || err != nil {
					log.Debug("Client Secret is invalid: ", clientID)
					gc.JSON(http.StatusBadRequest, gin.H{
						"error":             "invalid_client_secret",
						"error_description": "The client secret is invalid",
					})
					return
				}
			}

			// validate session
			claims, err := token.ValidateBrowserSession(gc, sessionDataSplit[1])
			if err != nil {
				log.Debug("Error validating session: ", err)
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": "Invalid session data",
				})
				return
			}

			userID = claims.Subject
			roles = claims.Roles
			scope = claims.Scope
			loginMethod = claims.LoginMethod

			// rollover the session for security
			sessionKey = userID
			if loginMethod != "" {
				sessionKey = loginMethod + ":" + userID
			}

			go memorystore.Provider.DeleteUserSession(sessionKey, claims.Nonce)

		} else {
			// validate refresh token
			if refreshToken == "" {
				log.Debug("Refresh token is empty")
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_refresh_token",
					"error_description": "The refresh token is invalid",
				})
				return
			}

			claims, err := token.ValidateRefreshToken(gc, refreshToken)
			if err != nil {
				log.Debug("Error validating refresh token: ", err)
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": err.Error(),
				})
				return
			}
			userID = claims["sub"].(string)
			claimLoginMethod := claims["login_method"]
			rolesInterface := claims["roles"].([]interface{})
			scopeInterface := claims["scope"].([]interface{})
			for _, v := range rolesInterface {
				roles = append(roles, v.(string))
			}
			for _, v := range scopeInterface {
				scope = append(scope, v.(string))
			}

			sessionKey = userID
			if claimLoginMethod != nil && claimLoginMethod != "" {
				sessionKey = claimLoginMethod.(string) + ":" + sessionKey
				loginMethod = claimLoginMethod.(string)
			}

			// remove older refresh token and rotate it for security
			go memorystore.Provider.DeleteUserSession(sessionKey, claims["nonce"].(string))
		}

		if sessionKey == "" {
			log.Debug("Error getting sessionKey: ", sessionKey, loginMethod)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User not found",
			})
			return
		}

		user, err := db.Provider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug("Error getting user: ", err)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User not found",
			})
			return
		}

		nonce := uuid.New().String() + "@@" + code
		authToken, err := token.CreateAuthToken(gc, user, roles, scope, loginMethod, nonce, code)
		if err != nil {
			log.Debug("Error creating auth token: ", err)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User not found",
			})
			return
		}

		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash)
		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token)
		cookie.SetSession(gc, authToken.FingerPrintHash)

		expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		if expiresIn <= 0 {
			expiresIn = 1
		}

		res := map[string]interface{}{
			"access_token": authToken.AccessToken.Token,
			"id_token":     authToken.IDToken.Token,
			"scope":        strings.Join(scope, " "),
			"roles":        roles,
			"expires_in":   expiresIn,
		}

		if authToken.RefreshToken != nil {
			res["refresh_token"] = authToken.RefreshToken.Token
			memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token)
		}

		gc.JSON(http.StatusOK, res)
	}
}
