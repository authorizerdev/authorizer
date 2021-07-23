package handlers

import (
	"net/http"

	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func OAuthLoginHandler(provider enum.OAuthProvider) gin.HandlerFunc {
	uuid := uuid.New()
	oauthStateString := uuid.String()

	return func(c *gin.Context) {
		if provider == enum.GoogleProvider {
			session.SetToken(oauthStateString, enum.Google.String())
			url := oauth.OAuthProvider.GoogleConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		}
		if provider == enum.GithubProvider {
			session.SetToken(oauthStateString, enum.Github.String())
			url := oauth.OAuthProvider.GithubConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		}
	}
}
