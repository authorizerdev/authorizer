package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// set host in the oauth state that is useful for redirecting

func OAuthLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO validate redirect URL
		redirectURL := c.Query("redirect_url")

		if redirectURL == "" {
			c.JSON(400, gin.H{
				"error": "invalid redirect url",
			})
			return
		}
		uuid := uuid.New()
		oauthStateString := uuid.String() + "___" + redirectURL

		provider := c.Param("oauth_provider")

		switch provider {
		case enum.Google.String():
			session.SetToken(oauthStateString, enum.Google.String())
			url := oauth.OAuthProvider.GoogleConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case enum.Github.String():
			session.SetToken(oauthStateString, enum.Github.String())
			url := oauth.OAuthProvider.GithubConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		default:
			c.JSON(422, gin.H{
				"message": "Invalid oauth provider",
			})
		}
	}
}
