package http_handlers

import (
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/gin-gonic/gin"
)

// RevokeRefreshTokenHandler handler to revoke refresh token
func (h *httpProvider) RevokeRefreshTokenHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "RevokeRefreshTokenHandler").Logger()
	return func(gc *gin.Context) {
		var reqBody map[string]string
		if err := gc.BindJSON(&reqBody); err != nil {
			log.Debug().Err(err).Msg("failed to bind json")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "error_binding_json",
				"error_description": err.Error(),
			})
			return
		}
		// get client ID
		clientID := strings.TrimSpace(reqBody["client_id"]) // kept for backward compatibility // else we expect to be present as header
		if clientID == "" {
			clientID = gc.Request.Header.Get("x-authorizer-client-id")
		}
		// get fingerprint hash
		refreshToken := strings.TrimSpace(reqBody["refresh_token"])

		if clientID == "" {
			log.Debug().Msg("Client ID is mising")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "client_id_required",
				"error_description": "The client id is missing",
			})
			return
		}

		if h.Config.ClientID != clientID {
			log.Debug().Str("client_id", clientID).Msg("Client ID is invalid")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_client_id",
				"error_description": "The client id is invalid",
			})
			return
		}

		claims, err := h.TokenProvider.ParseJWTToken(refreshToken)
		if err != nil {
			log.Debug().Err(err).Msg("failed to parse jwt")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             err.Error(),
				"error_description": "Failed to parse jwt",
			})
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			log.Debug().Msg("Invalid subject in refresh token")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_token",
				"error_description": "Invalid refresh token",
			})
			return
		}

		loginMethod := claims["login_method"]
		sessionToken := userID
		if lm, ok := loginMethod.(string); ok && lm != "" {
			sessionToken = lm + ":" + userID
		}

		nonce, ok := claims["nonce"].(string)
		if !ok || nonce == "" {
			log.Debug().Msg("Invalid nonce in refresh token")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_token",
				"error_description": "Invalid refresh token",
			})
			return
		}

		existingToken, err := h.MemoryStoreProvider.GetUserSession(sessionToken, constants.TokenTypeRefreshToken+"_"+nonce)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get refresh token")
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error":             "failed_to_get_refresh_token",
				"error_description": "Failed to get user refresh token: " + err.Error(),
			})
			return
		}

		if existingToken == "" {
			log.Debug().Msg("Token not found")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "token_not_found",
				"error_description": "Token not found",
			})
			return
		}

		if existingToken != refreshToken {
			log.Debug().Msg("Token does not match")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "token_does_not_match",
				"error_description": "Token does not match",
			})
			return
		}

		if err := h.MemoryStoreProvider.DeleteUserSession(sessionToken, nonce); err != nil {
			log.Debug().Err(err).Msg("failed to delete user session")
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error":             "failed_to_delete_user_session",
				"error_description": "Failed to delete user session: " + err.Error(),
			})
			return
		}

		gc.JSON(http.StatusOK, gin.H{
			"message": "Token revoked successfully",
		})
	}
}
