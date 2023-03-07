package handlers

import (
	"encoding/json"
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
		user, err := db.Provider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug("Error getting user: ", err)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		apiUser := user.AsAPIUser()
		userBytes, err := json.Marshal(apiUser)
		if err != nil {
			log.Debug("Error marshalling user: ", err)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		res := map[string]interface{}{}
		err = json.Unmarshal(userBytes, &res)
		if err != nil {
			log.Debug("Error un-marshalling user: ", err)
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
