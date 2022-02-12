package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
)

// VerifyEmailHandler handles the verify email route.
// It verifies email based on JWT token in query string
func VerifyEmailHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		errorRes := gin.H{
			"message": "invalid token",
		}
		tokenInQuery := c.Query("token")
		if tokenInQuery == "" {
			c.JSON(400, errorRes)
			return
		}

		verificationRequest, err := db.Provider.GetVerificationRequestByToken(tokenInQuery)
		if err != nil {
			c.JSON(400, errorRes)
			return
		}

		// verify if token exists in db
		claim, err := token.ParseJWTToken(tokenInQuery)
		if err != nil {
			c.JSON(400, errorRes)
			return
		}

		user, err := db.Provider.GetUserByEmail(claim["email"].(string))
		if err != nil {
			c.JSON(400, gin.H{
				"message": err.Error(),
			})
			return
		}

		// update email_verified_at in users table
		if user.EmailVerifiedAt == nil {
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
			db.Provider.UpdateUser(user)
		}
		// delete from verification table
		db.Provider.DeleteVerificationRequest(verificationRequest)

		roles := strings.Split(user.Roles, ",")
		authToken, err := token.CreateAuthToken(user, roles)
		if err != nil {
			c.JSON(400, gin.H{
				"message": err.Error(),
			})
			return
		}
		sessionstore.SetUserSession(user.ID, authToken.FingerPrint, authToken.RefreshToken.Token)
		cookie.SetCookie(c, authToken.AccessToken.Token, authToken.RefreshToken.Token, authToken.FingerPrintHash)
		utils.SaveSessionInDB(user.ID, c)

		c.Redirect(http.StatusTemporaryRedirect, claim["redirect_url"].(string))
	}
}
