package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yauthdev/yauth/server/enum"
	"github.com/yauthdev/yauth/server/oauth"
	"github.com/yauthdev/yauth/server/session"
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
