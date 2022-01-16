package handlers

import (
	"net/http"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OAuthLoginHandler set host in the oauth state that is useful for redirecting to oauth_callback
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
			if !utils.IsValidRoles(append([]string{}, append(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyRoles).([]string), envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyProtectedRoles).([]string)...)...), rolesSplit) {
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
		} else {
			roles = strings.Join(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDefaultRoles).([]string), ",")
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
			session.SetSocailLoginState(oauthStateString, constants.SignupMethodGoogle)
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProviders.GoogleConfig.RedirectURL = envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAuthorizerURL).(string) + "/oauth_callback/google"
			url := oauth.OAuthProviders.GoogleConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.SignupMethodGithub:
			if oauth.OAuthProviders.GithubConfig == nil {
				isProviderConfigured = false
				break
			}
			session.SetSocailLoginState(oauthStateString, constants.SignupMethodGithub)
			oauth.OAuthProviders.GithubConfig.RedirectURL = envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAuthorizerURL).(string) + "/oauth_callback/github"
			url := oauth.OAuthProviders.GithubConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.SignupMethodFacebook:
			if oauth.OAuthProviders.FacebookConfig == nil {
				isProviderConfigured = false
				break
			}
			session.SetSocailLoginState(oauthStateString, constants.SignupMethodFacebook)
			oauth.OAuthProviders.FacebookConfig.RedirectURL = envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAuthorizerURL).(string) + "/oauth_callback/facebook"
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
