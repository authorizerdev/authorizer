package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerifyEmailHandler handles the verify email route.
// It verifies email based on JWT token in query string
func VerifyEmailHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		errorRes := gin.H{
			"error": "invalid_token",
		}
		tokenInQuery := c.Query("token")
		if tokenInQuery == "" {
			log.Debug("Token is empty")
			c.JSON(400, errorRes)
			return
		}

		verificationRequest, err := db.Provider.GetVerificationRequestByToken(c, tokenInQuery)
		if err != nil {
			log.Debug("Error getting verification request: ", err)
			errorRes["error_description"] = err.Error()
			c.JSON(400, errorRes)
			return
		}

		// verify if token exists in db
		hostname := parsers.GetHost(c)
		claim, err := token.ParseJWTToken(tokenInQuery)
		if err != nil {
			log.Debug("Error parsing token: ", err)
			errorRes["error_description"] = err.Error()
			c.JSON(400, errorRes)
			return
		}

		if ok, err := token.ValidateJWTClaims(claim, hostname, verificationRequest.Nonce, verificationRequest.Email); !ok || err != nil {
			log.Debug("Error validating jwt claims: ", err)
			errorRes["error_description"] = err.Error()
			c.JSON(400, errorRes)
			return
		}

		user, err := db.Provider.GetUserByEmail(c, verificationRequest.Email)
		if err != nil {
			log.Debug("Error getting user: ", err)
			errorRes["error_description"] = err.Error()
			c.JSON(400, errorRes)
			return
		}

		isSignUp := false
		// update email_verified_at in users table
		if user.EmailVerifiedAt == nil {
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
			isSignUp = true
			db.Provider.UpdateUser(c, user)
		}
		// delete from verification table
		db.Provider.DeleteVerificationRequest(c, verificationRequest)

		state := strings.TrimSpace(c.Query("state"))
		redirectURL := strings.TrimSpace(c.Query("redirect_uri"))
		rolesString := strings.TrimSpace(c.Query("roles"))
		var roles []string
		if rolesString == "" {
			roles = strings.Split(user.Roles, ",")
		} else {
			roles = strings.Split(rolesString, ",")
		}

		scopeString := strings.TrimSpace(c.Query("scope"))
		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "email", "profile"}
		} else {
			scope = strings.Split(scopeString, " ")
		}
		loginMethod := constants.AuthRecipeMethodBasicAuth
		if verificationRequest.Identifier == constants.VerificationTypeMagicLinkLogin {
			loginMethod = constants.AuthRecipeMethodMagicLinkLogin
		}

		nonce := uuid.New().String()
		authToken, err := token.CreateAuthToken(c, user, roles, scope, loginMethod, nonce)
		if err != nil {
			log.Debug("Error creating auth token: ", err)
			errorRes["error_description"] = err.Error()
			c.JSON(500, errorRes)
			return
		}

		expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		if expiresIn <= 0 {
			expiresIn = 1
		}

		params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + state + "&id_token=" + authToken.IDToken.Token

		sessionKey := loginMethod + ":" + user.ID
		cookie.SetSession(c, authToken.FingerPrintHash)
		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash)
		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token)

		if authToken.RefreshToken != nil {
			params = params + `&refresh_token=` + authToken.RefreshToken.Token
			memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token)
		}

		if redirectURL == "" {
			redirectURL = claim["redirect_uri"].(string)
		}

		if strings.Contains(redirectURL, "?") {
			redirectURL = redirectURL + "&" + params
		} else {
			redirectURL = redirectURL + "?" + strings.TrimPrefix(params, "&")
		}

		go func() {
			if isSignUp {
				utils.RegisterEvent(c, constants.UserSignUpWebhookEvent, loginMethod, user)
			} else {
				utils.RegisterEvent(c, constants.UserLoginWebhookEvent, loginMethod, user)
			}

			db.Provider.AddSession(c, models.Session{
				UserID:    user.ID,
				UserAgent: utils.GetUserAgent(c.Request),
				IP:        utils.GetIP(c.Request),
			})
		}()

		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}
