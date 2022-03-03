package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/gin-gonic/gin"
)

func LogoutHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		// get fingerprint hash
		fingerprintHash, err := cookie.GetSession(gc)
		if err != nil {
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		decryptedFingerPrint, err := crypto.DecryptAES(fingerprintHash)
		if err != nil {
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		fingerPrint := string(decryptedFingerPrint)

		sessionstore.RemoveState(fingerPrint)
		cookie.DeleteSession(gc)

		gc.JSON(http.StatusOK, gin.H{
			"message": "Logged out successfully",
		})
	}
}
