package http_handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/validators"
)

// OAuthLoginHandler set host in the oauth state that is useful for redirecting to oauth_callback
func (h *httpProvider) OAuthLoginHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "OAuthLoginHandler").Logger()
	return func(c *gin.Context) {
		// deprecating redirectURL instead use redirect_uri
		redirectURI := strings.TrimSpace(c.Query("redirectURL"))
		if redirectURI == "" {
			redirectURI = strings.TrimSpace(c.Query("redirect_uri"))
		}
		roles := strings.TrimSpace(c.Query("roles"))
		state := strings.TrimSpace(c.Query("state"))
		scopeString := strings.TrimSpace(c.Query("scope"))

		if redirectURI == "" {
			log.Debug().Msg("redirect uri is missing")
			c.JSON(400, gin.H{
				"error": "invalid redirect uri",
			})
			return
		}

		if state == "" {
			log.Debug().Msg("state is missing, creating new state")
			state = uuid.New().String()
		}

		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "profile", "email"}
		} else {
			scope = strings.Split(scopeString, " ")
		}

		if roles != "" {
			// validate role
			rolesSplit := strings.Split(roles, ",")

			// use protected roles verification for admin login only.
			// though if not associated with user, it will be rejected from oauth_callback
			allowedRoles := h.Config.Roles
			protectedRoles := h.Config.ProtectedRoles
			if !validators.IsValidRoles(rolesSplit, append([]string{}, append(allowedRoles, protectedRoles...)...)) {
				log.Debug().Msg("invalid role")
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
		} else {
			roles = strings.Join(h.Config.DefaultRoles, ",")
		}

		oauthStateString := state + "___" + redirectURI + "___" + roles + "___" + strings.Join(scope, " ")

		provider := c.Param("oauth_provider")
		log := log.With().Str("provider", provider).Logger()
		cfg, err := h.OAuthProvider.GetOAuthConfig(c, provider)
		if err != nil {
			log.Debug().Err(err).Msg("Error getting oauth config")
			c.JSON(422, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := h.MemoryStoreProvider.SetState(oauthStateString, provider); err != nil {
			log.Debug().Err(err).Msg("Error setting state")
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}
		url := cfg.AuthCodeURL(oauthStateString)
		log.Debug().Str("url", url).Msg("redirecting to oauth provider")
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}
