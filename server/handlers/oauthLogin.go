package handlers

import (
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// set host in the oauth state that is useful for redirecting

func OAuthLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO validate redirect URL
		redirectURL := c.Query("redirectURL")
		roles := c.Query("roles")

		if redirectURL == "" {
			c.JSON(400, gin.H{
				"error": "invalid redirect url",
			})
			return
		}

		if roles != "" {
			// validate role
			rolesSplit := strings.Split(roles, ",")

			// use protected roles verification for admin login only.
			// though if not associated with user, it will be rejected from oauth_callback
			if !utils.IsValidRoles(append([]string{}, append(constants.ROLES, constants.PROTECTED_ROLES...)...), rolesSplit) {
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
		} else {
			roles = strings.Join(constants.DEFAULT_ROLES, ",")
		}

		uuid := uuid.New()
		oauthStateString := uuid.String() + "___" + redirectURL + "___" + roles

		provider := c.Param("oauth_provider")

		switch provider {
		case enum.Google.String():
			session.SetToken(oauthStateString, enum.Google.String())
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProvider.GoogleConfig.RedirectURL = constants.AUTHORIZER_URL + "/oauth_callback/google"
			url := oauth.OAuthProvider.GoogleConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case enum.Github.String():
			session.SetToken(oauthStateString, enum.Github.String())
			oauth.OAuthProvider.GithubConfig.RedirectURL = constants.AUTHORIZER_URL + "/oauth_callback/github"
			url := oauth.OAuthProvider.GithubConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case enum.Facebook.String():
			session.SetToken(oauthStateString, enum.Github.String())
			oauth.OAuthProvider.FacebookConfig.RedirectURL = constants.AUTHORIZER_URL + "/oauth_callback/facebook"
			url := oauth.OAuthProvider.FacebookConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		default:
			c.JSON(422, gin.H{
				"message": "Invalid oauth provider",
			})
		}
	}
}
