package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/token"
)

func UserInfoHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		accessToken, err := token.GetAccessToken(gc)
		if err != nil {
			log.Debug("Error getting access token: ", err)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		claims, err := token.ValidateAccessToken(gc, accessToken)
		if err != nil {
			log.Debug("Error validating access token: ", err)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		userID := claims["sub"].(string)
		user, err := db.Provider.GetUserByID(userID)
		if err != nil {
			log.Debug("Error getting user: ", err)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		gc.JSON(http.StatusOK, user.AsAPIUser())
	}
}
