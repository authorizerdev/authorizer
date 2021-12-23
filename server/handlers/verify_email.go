package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
)

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

		verificationRequest, err := db.Mgr.GetVerificationByToken(token)
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

		user, err := db.Mgr.GetUserByEmail(claim.Email)
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
			db.Mgr.UpdateUser(user)
		}
		// delete from verification table
		db.Mgr.DeleteVerificationRequest(verificationRequest)

		roles := strings.Split(user.Roles, ",")
		refreshToken, _, _ := utils.CreateAuthToken(user, enum.RefreshToken, roles)

		accessToken, _, _ := utils.CreateAuthToken(user, enum.AccessToken, roles)

		session.SetToken(user.ID, accessToken, refreshToken)
		utils.CreateSession(user.ID, c)
		utils.SetCookie(c, accessToken)
		c.Redirect(http.StatusTemporaryRedirect, claim.RedirectURL)
	}
}
