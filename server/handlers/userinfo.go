package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/gin-gonic/gin"
)

func UserInfoHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		accessToken, err := token.GetAccessToken(gc)
		if err != nil {
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		claims, err := token.ValidateAccessToken(gc, accessToken)
		if err != nil {
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		userID := claims["sub"].(string)
		user, err := db.Provider.GetUserByID(userID)
		if err != nil {
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		gc.JSON(http.StatusOK, user.AsAPIUser())
	}
}
