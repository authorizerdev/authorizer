package http_handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/token"
)

// Handler to logout user
func (h *httpProvider) LogoutHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "LogoutHandler").Logger()
	return func(gc *gin.Context) {
		redirectURL := strings.TrimSpace(gc.Query("redirect_uri"))
		// get fingerprint hash
		fingerprintHash, err := cookie.GetSession(gc)
		if err != nil {
			log.Debug().Err(err).Msg("failed GetSession")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		decryptedFingerPrint, err := crypto.DecryptAES(h.ClientSecret, fingerprintHash)
		if err != nil {
			log.Debug().Err(err).Msg("failed to decrypt fingerprint")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		var sessionData token.SessionData
		err = json.Unmarshal([]byte(decryptedFingerPrint), &sessionData)
		if err != nil {
			log.Debug().Err(err).Msg("failed to unmarshal session data")
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}

		userID := sessionData.Subject
		loginMethod := sessionData.LoginMethod
		sessionToken := userID
		if loginMethod != "" {
			sessionToken = loginMethod + ":" + userID
		}

		h.MemoryStoreProvider.DeleteUserSession(sessionToken, sessionData.Nonce)
		cookie.DeleteSession(gc)

		if redirectURL != "" {
			gc.Redirect(http.StatusFound, redirectURL)
		} else {
			gc.JSON(http.StatusOK, gin.H{
				"message": "Logged out successfully",
			})
		}
	}
}
