package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/utils"
)

// OAuthLoginHandler set host in the oauth state that is useful for redirecting to oauth_callback
func OAuthLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		hostname := utils.GetHost(c)
		// deprecating redirectURL instead use redirect_uri
		redirectURI := strings.TrimSpace(c.Query("redirectURL"))
		if redirectURI == "" {
			redirectURI = strings.TrimSpace(c.Query("redirect_uri"))
		}
		roles := strings.TrimSpace(c.Query("roles"))
		state := strings.TrimSpace(c.Query("state"))
		scopeString := strings.TrimSpace(c.Query("scope"))

		if redirectURI == "" {
			log.Debug("redirect_uri is empty")
			c.JSON(400, gin.H{
				"error": "invalid redirect uri",
			})
			return
		}

		if state == "" {
			log.Debug("state is empty")
			c.JSON(400, gin.H{
				"error": "invalid state",
			})
			return
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
			roles, err := memorystore.Provider.GetSliceStoreEnvVariable(constants.EnvKeyRoles)
			if err != nil {
				log.Debug("Error getting roles: ", err)
			}
			protectedRoles, err := memorystore.Provider.GetSliceStoreEnvVariable(constants.EnvKeyProtectedRoles)
			if err != nil {
				log.Debug("Error getting protected roles: ", err)
			}

			if !utils.IsValidRoles(rolesSplit, append([]string{}, append(roles, protectedRoles...)...)) {
				log.Debug("Invalid roles: ", roles)
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
		} else {
			defaultRoles, err := memorystore.Provider.GetSliceStoreEnvVariable(constants.EnvKeyDefaultRoles)
			if err != nil {
				log.Debug("Error getting default roles: ", err)
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
			roles = strings.Join(defaultRoles, ",")

		}

		oauthStateString := state + "___" + redirectURI + "___" + roles + "___" + strings.Join(scope, ",")

		provider := c.Param("oauth_provider")
		isProviderConfigured := true
		switch provider {
		case constants.SignupMethodGoogle:
			if oauth.OAuthProviders.GoogleConfig == nil {
				log.Debug("Google OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.SignupMethodGoogle)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProviders.GoogleConfig.RedirectURL = hostname + "/oauth_callback/google"
			url := oauth.OAuthProviders.GoogleConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.SignupMethodGithub:
			if oauth.OAuthProviders.GithubConfig == nil {
				log.Debug("Github OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.SignupMethodGithub)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.GithubConfig.RedirectURL = hostname + "/oauth_callback/github"
			url := oauth.OAuthProviders.GithubConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.SignupMethodFacebook:
			if oauth.OAuthProviders.FacebookConfig == nil {
				log.Debug("Facebook OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.SignupMethodFacebook)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.FacebookConfig.RedirectURL = hostname + "/oauth_callback/facebook"
			url := oauth.OAuthProviders.FacebookConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		default:
			log.Debug("Invalid oauth provider: ", provider)
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
