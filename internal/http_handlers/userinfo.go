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
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		claims, err := h.TokenProvider.ValidateAccessToken(gc, accessToken)
		if err != nil {
			log.Debug().Msg("Error validating access token")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		userID := claims["sub"].(string)
		user, err := h.StorageProvider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug().Msg("Error getting user by ID")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		apiUser := user.AsAPIUser()
		userBytes, err := json.Marshal(apiUser)
		if err != nil {
			log.Debug().Msg("Error marshalling user")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		res := map[string]interface{}{}
		err = json.Unmarshal(userBytes, &res)
		if err != nil {
			log.Debug().Msg("Error unmarshalling user")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		// add sub field to user as per openid standards
		// https://github.com/authorizerdev/authorizer/issues/327
		res["sub"] = userID
		gc.JSON(http.StatusOK, res)
	}
}
