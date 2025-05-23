package http_handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// State is the struct that holds authorizer url and redirect url
// They are provided via query string in the request
type State struct {
	AuthorizerURL string `json:"authorizerURL"`
	RedirectURL   string `json:"redirectURL"`
}

// AppHandler is the handler for the /app route
func (h *httpProvider) AppHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "AppHandler").Logger()
	return func(c *gin.Context) {
		hostname := parsers.GetHost(c)
		if h.Config.DisableLoginPage {
			log.Debug().Msg("Login page is disabled")
			c.JSON(400, gin.H{"error": "login page is not enabled"})
			return
		}

		redirectURI := strings.TrimSpace(c.Query("redirect_uri"))
		state := strings.TrimSpace(c.Query("state"))
		scopeString := strings.TrimSpace(c.Query("scope"))

		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "profile", "email"}
		} else {
			scope = strings.Split(scopeString, " ")
		}

		if redirectURI == "" {
			redirectURI = hostname + "/app"
		} else {
			// validate redirect url with allowed origins
			if !validators.IsValidOrigin(redirectURI, h.Config.AllowedOrigins) {
				log.Debug().Msg("Invalid redirect url")
				c.JSON(400, gin.H{"error": "invalid redirect url"})
				return
			}
		}

		// debug the request state
		if pusher := c.Writer.Pusher(); pusher != nil {
			// use pusher.Push() to do server push
			if err := pusher.Push("/app/build/bundle.js", nil); err != nil {
				log.Debug().Err(err).Msg("Failed to push bundle.js")
			}
		}

		orgName := h.Config.OrganizationName
		orgLogo := h.Config.OrganizationLogo
		c.HTML(http.StatusOK, "app.tmpl", gin.H{
			"data": map[string]interface{}{
				"authorizerURL":    hostname,
				"redirectURL":      redirectURI,
				"scope":            strings.Join(scope, " "),
				"state":            state,
				"organizationName": orgName,
				"organizationLogo": orgLogo,
			},
		})
	}
}
