package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func OAuthLoginHandler() gin.HandlerFunc {
	uuid := uuid.New()
	oauthStateString := uuid.String()

	return func(c *gin.Context) {
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
