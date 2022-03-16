package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/gin-gonic/gin"
)

// TokenHandler to handle /oauth/token requests
// grant type required
func TokenHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		var reqBody map[string]string
		if err := gc.BindJSON(&reqBody); err != nil {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "error_binding_json",
				"error_description": err.Error(),
			})
			return
		}

		codeVerifier := strings.TrimSpace(reqBody["code_verifier"])
		code := strings.TrimSpace(reqBody["code"])
		clientID := strings.TrimSpace(reqBody["client_id"])
		grantType := strings.TrimSpace(reqBody["grant_type"])
		refreshToken := strings.TrimSpace(reqBody["refresh_token"])

		if grantType == "" {
			grantType = "authorization_code"
		}

		isRefreshTokenGrant := grantType == "refresh_token"
		isAuthorizationCodeGrant := grantType == "authorization_code"

		if !isRefreshTokenGrant && !isAuthorizationCodeGrant {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_grant_type",
				"error_description": "grant_type is invalid",
			})
		}

		if clientID == "" {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "client_id_required",
				"error_description": "The client id is required",
			})
			return
		}

		if clientID != envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID) {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client_id",
				"error_description": "The client id is invalid",
			})
			return
		}

		var userID string
		var roles, scope []string
		if isAuthorizationCodeGrant {

			if codeVerifier == "" {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_code_verifier",
					"error_description": "The code verifier is required",
				})
				return
			}

			if code == "" {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_code",
					"error_description": "The code is required",
				})
				return
			}

			hash := sha256.New()
			hash.Write([]byte(codeVerifier))
			encryptedCode := strings.ReplaceAll(base64.URLEncoding.EncodeToString(hash.Sum(nil)), "+", "-")
			encryptedCode = strings.ReplaceAll(encryptedCode, "/", "_")
			encryptedCode = strings.ReplaceAll(encryptedCode, "=", "")
			sessionData := sessionstore.GetState(encryptedCode)
			if sessionData == "" {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_code_verifier",
					"error_description": "The code verifier is invalid",
				})
				return
			}

			// split session data
			// it contains code@sessiontoken
			sessionDataSplit := strings.Split(sessionData, "@")

			if sessionDataSplit[0] != code {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_code_verifier",
					"error_description": "The code verifier is invalid",
				})
				return
			}

			// validate session
			claims, err := token.ValidateBrowserSession(gc, sessionDataSplit[1])
			if err != nil {
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": "Invalid session data",
				})
				return
			}
			// rollover the session for security
			sessionstore.RemoveState(sessionDataSplit[1])
			userID = claims.Subject
			roles = claims.Roles
			scope = claims.Scope
		} else {
			// validate refresh token
			if refreshToken == "" {
				gc.JSON(http.StatusBadRequest, gin.H{
					"error":             "invalid_refresh_token",
					"error_description": "The refresh token is invalid",
				})
			}

			claims, err := token.ValidateRefreshToken(gc, refreshToken)
			if err != nil {
				gc.JSON(http.StatusUnauthorized, gin.H{
					"error":             "unauthorized",
					"error_description": err.Error(),
				})
			}
			userID = claims["sub"].(string)
			rolesInterface := claims["roles"].([]interface{})
			scopeInterface := claims["scope"].([]interface{})
			for _, v := range rolesInterface {
				roles = append(roles, v.(string))
			}
			for _, v := range scopeInterface {
				scope = append(scope, v.(string))
			}
			// remove older refresh token and rotate it for security
			sessionstore.RemoveState(refreshToken)
		}

		user, err := db.Provider.GetUserByID(userID)
		if err != nil {
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User not found",
			})
			return
		}

		authToken, err := token.CreateAuthToken(gc, user, roles, scope)
		if err != nil {
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "unauthorized",
				"error_description": "User not found",
			})
			return
		}
		sessionstore.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
		sessionstore.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)
		cookie.SetSession(gc, authToken.FingerPrintHash)

		expiresIn := int64(1800)
		res := map[string]interface{}{
			"access_token": authToken.AccessToken.Token,
			"id_token":     authToken.IDToken.Token,
			"scope":        scope,
			"roles":        roles,
			"expires_in":   expiresIn,
		}

		if authToken.RefreshToken != nil {
			res["refresh_token"] = authToken.RefreshToken.Token
			sessionstore.SetState(authToken.RefreshToken.Token, authToken.FingerPrint+"@"+user.ID)
		}

		gc.JSON(http.StatusOK, res)
	}
}
