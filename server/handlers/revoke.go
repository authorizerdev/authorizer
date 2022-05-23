package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/sessionstore"
)

// Revoke handler to revoke refresh token
func RevokeHandler() gin.HandlerFunc {
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

		if clientID != envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID) {
			log.Debug("Client ID is invalid")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client_id",
				"error_description": "The client id is invalid",
			})
			return
		}

		sessionstore.RemoveState(refreshToken)

		gc.JSON(http.StatusOK, gin.H{
			"message": "Token revoked successfully",
		})
	}
}
