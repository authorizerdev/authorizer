package handlers

import (
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OAuthLoginHandler set host in the oauth state that is useful for redirecting to oauth_callback
func OAuthLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		hostname := utils.GetHost(c)
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
			if !utils.IsValidRoles(append([]string{}, append(envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyRoles), envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyProtectedRoles)...)...), rolesSplit) {
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
		} else {
			roles = strings.Join(envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyDefaultRoles), ",")
		}

		uuid := uuid.New()
		oauthStateString := uuid.String() + "___" + redirectURL + "___" + roles

		provider := c.Param("oauth_provider")
		isProviderConfigured := true
		switch provider {
		case constants.SignupMethodGoogle:
			if oauth.OAuthProviders.GoogleConfig == nil {
				isProviderConfigured = false
				break
			}
			sessionstore.SetState(oauthStateString, constants.SignupMethodGoogle)
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProviders.GoogleConfig.RedirectURL = hostname + "/oauth_callback/google"
			url := oauth.OAuthProviders.GoogleConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.SignupMethodGithub:
			if oauth.OAuthProviders.GithubConfig == nil {
				isProviderConfigured = false
				break
			}
			sessionstore.SetState(oauthStateString, constants.SignupMethodGithub)
			oauth.OAuthProviders.GithubConfig.RedirectURL = hostname + "/oauth_callback/github"
			url := oauth.OAuthProviders.GithubConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.SignupMethodFacebook:
			if oauth.OAuthProviders.FacebookConfig == nil {
				isProviderConfigured = false
				break
			}
			sessionstore.SetState(oauthStateString, constants.SignupMethodFacebook)
			oauth.OAuthProviders.FacebookConfig.RedirectURL = hostname + "/oauth_callback/facebook"
			url := oauth.OAuthProviders.FacebookConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		default:
			c.JSON(422, gin.H{
				"message": "Invalid oauth provider",
			})
		}

		if !isProviderConfigured {
			c.JSON(422, gin.H{
				"message": provider + " not configured",
			})
		}
	}
}
