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
	"github.com/google/uuid"
)

// VerifyEmailHandler handles the verify email route.
// It verifies email based on JWT token in query string
func VerifyEmailHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		errorRes := gin.H{
			"error": "invalid token",
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
		hostname := utils.GetHost(c)
		encryptedNonce, err := utils.EncryptNonce(verificationRequest.Nonce)
		if err != nil {
			c.JSON(400, gin.H{
				"error": err.Error(),
			})
			return
		}
		claim, err := token.ParseJWTToken(tokenInQuery, hostname, encryptedNonce, verificationRequest.Email)
		if err != nil {
			c.JSON(400, errorRes)
			return
		}

		user, err := db.Provider.GetUserByEmail(claim["sub"].(string))
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
		scope := []string{"openid", "email", "profile"}
		nonce := uuid.New().String()
		_, authToken, err := token.CreateSessionToken(user, nonce, roles, scope)
		if err != nil {
			c.JSON(400, gin.H{
				"message": err.Error(),
			})
			return
		}
		sessionstore.SetState(authToken, nonce+"@"+user.ID)
		cookie.SetSession(c, authToken)

		go utils.SaveSessionInDB(c, user.ID)

		c.Redirect(http.StatusTemporaryRedirect, claim["redirect_url"].(string))
	}
}
