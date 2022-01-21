package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/session"
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
		token := c.Query("token")
		if token == "" {
			c.JSON(400, errorRes)
			return
		}

		verificationRequest, err := db.Provider.GetVerificationRequestByToken(token)
		if err != nil {
			c.JSON(400, errorRes)
			return
		}

		// verify if token exists in db
		claim, err := utils.VerifyVerificationToken(token)
		if err != nil {
			c.JSON(400, errorRes)
			return
		}

		user, err := db.Provider.GetUserByEmail(claim.Email)
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
		refreshToken, _, _ := utils.CreateAuthToken(user, constants.TokenTypeRefreshToken, roles)

		accessToken, _, _ := utils.CreateAuthToken(user, constants.TokenTypeAccessToken, roles)

		session.SetUserSession(user.ID, accessToken, refreshToken)
		utils.SaveSessionInDB(user.ID, c)
		utils.SetCookie(c, accessToken)
		c.Redirect(http.StatusTemporaryRedirect, claim.RedirectURL)
	}
}
