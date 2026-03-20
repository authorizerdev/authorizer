package http_handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *httpProvider) UserInfoHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "UserInfoHandler").Logger()
	return func(gc *gin.Context) {
		accessToken, err := h.TokenProvider.GetAccessToken(gc)
		if err != nil {
			log.Debug().Msg("Error getting access token")
			// RFC 6750 §3: No credentials - return 401 with WWW-Authenticate
			gc.Header("WWW-Authenticate", `Bearer realm="authorizer"`)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_request",
				"error_description": "No access token provided",
			})
			return
		}
		claims, err := h.TokenProvider.ValidateAccessToken(gc, accessToken)
		if err != nil {
			log.Debug().Msg("Error validating access token")
			// RFC 6750 §3.1: Invalid token - return 401 with error details
			gc.Header("WWW-Authenticate", `Bearer realm="authorizer", error="invalid_token", error_description="The access token is invalid or has expired"`)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "The access token is invalid or has expired",
			})
			return
		}
		userID := claims["sub"].(string)
		user, err := h.StorageProvider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug().Msg("Error getting user by ID")
			gc.Header("WWW-Authenticate", `Bearer realm="authorizer", error="invalid_token", error_description="The user associated with this token was not found"`)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "The user associated with this token was not found",
			})
			return
		}
		apiUser := user.AsAPIUser()
		userBytes, err := json.Marshal(apiUser)
		if err != nil {
			log.Debug().Msg("Error marshalling user")
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error":             "server_error",
				"error_description": "Failed to process user data",
			})
			return
		}
		res := map[string]interface{}{}
		err = json.Unmarshal(userBytes, &res)
		if err != nil {
			log.Debug().Msg("Error unmarshalling user")
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error":             "server_error",
				"error_description": "Failed to process user data",
			})
			return
		}
		// OIDC Core §5.3.2: sub claim MUST always be returned
		res["sub"] = userID
		gc.JSON(http.StatusOK, res)
	}
}
