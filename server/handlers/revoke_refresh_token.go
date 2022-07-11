package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
)

// RevokeRefreshTokenHandler handler to revoke refresh token
func RevokeRefreshTokenHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		var reqBody map[string]string
		if err := gc.BindJSON(&reqBody); err != nil {
			log.Debug("Error binding JSON: ", err)
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "error_binding_json",
				"error_description": err.Error(),
			})
			return
		}
		// get fingerprint hash
		refreshToken := strings.TrimSpace(reqBody["refresh_token"])
		clientID := strings.TrimSpace(reqBody["client_id"])

		if clientID == "" {
			log.Debug("Client ID is empty")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "client_id_required",
				"error_description": "The client id is required",
			})
			return
		}

		if client, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID); client != clientID || err != nil {
			log.Debug("Client ID is invalid: ", clientID)
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client_id",
				"error_description": "The client id is invalid",
			})
			return
		}

		claims, err := token.ParseJWTToken(refreshToken)
		if err != nil {
			log.Debug("Client ID is invalid: ", clientID)
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             err.Error(),
				"error_description": "Failed to parse jwt",
			})
			return
		}

		userID := claims["sub"].(string)
		loginMethod := claims["login_method"]
		sessionToken := userID
		if loginMethod != nil && loginMethod != "" {
			sessionToken = loginMethod.(string) + ":" + userID
		}

		memorystore.Provider.DeleteUserSession(sessionToken, claims["nonce"].(string))

		gc.JSON(http.StatusOK, gin.H{
			"message": "Token revoked successfully",
		})
	}
}
