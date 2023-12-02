package handlers

import (
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
)

// OAuthLoginHandler set host in the oauth state that is useful for redirecting to oauth_callback
func OAuthLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		hostname := parsers.GetHost(c)
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
			rolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyRoles)
			roles := []string{}
			if err != nil {
				log.Debug("Error getting roles: ", err)
				rolesString = ""
			} else {
				roles = strings.Split(rolesString, ",")
			}

			protectedRolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyProtectedRoles)
			protectedRoles := []string{}
			if err != nil {
				log.Debug("Error getting protected roles: ", err)
				protectedRolesString = ""
			} else {
				protectedRoles = strings.Split(protectedRolesString, ",")
			}

			if !validators.IsValidRoles(rolesSplit, append([]string{}, append(roles, protectedRoles...)...)) {
				log.Debug("Invalid roles: ", roles)
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
		} else {
			defaultRoles, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
			if err != nil {
				log.Debug("Error getting default roles: ", err)
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
			roles = defaultRoles

		}

		oauthStateString := state + "___" + redirectURI + "___" + roles + "___" + strings.Join(scope, " ")

		provider := c.Param("oauth_provider")
		isProviderConfigured := true
		switch provider {
		case constants.AuthRecipeMethodGoogle:
			if oauth.OAuthProviders.GoogleConfig == nil {
				log.Debug("Google OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.AuthRecipeMethodGoogle)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProviders.GoogleConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodGoogle
			url := oauth.OAuthProviders.GoogleConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodGithub:
			if oauth.OAuthProviders.GithubConfig == nil {
				log.Debug("Github OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.AuthRecipeMethodGithub)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.GithubConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodGithub
			url := oauth.OAuthProviders.GithubConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodFacebook:
			if oauth.OAuthProviders.FacebookConfig == nil {
				log.Debug("Facebook OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.AuthRecipeMethodFacebook)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.FacebookConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodFacebook
			url := oauth.OAuthProviders.FacebookConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodLinkedIn:
			if oauth.OAuthProviders.LinkedInConfig == nil {
				log.Debug("Linkedin OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.AuthRecipeMethodLinkedIn)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.LinkedInConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodLinkedIn
			url := oauth.OAuthProviders.LinkedInConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodTwitter:
			if oauth.OAuthProviders.TwitterConfig == nil {
				log.Debug("Twitter OAuth provider is not configured")
				isProviderConfigured = false
				break
			}

			verifier, challenge := utils.GenerateCodeChallenge()

			err := memorystore.Provider.SetState(oauthStateString, verifier)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.TwitterConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodTwitter
			url := oauth.OAuthProviders.TwitterConfig.AuthCodeURL(oauthStateString, oauth2.SetAuthURLParam("code_challenge", challenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodApple:
			if oauth.OAuthProviders.AppleConfig == nil {
				log.Debug("Apple OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.AuthRecipeMethodApple)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.AppleConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodApple
			// there is scope encoding issue with oauth2 and how apple expects, hence added scope manually
			// check: https://github.com/golang/oauth2/issues/449
			url := oauth.OAuthProviders.AppleConfig.AuthCodeURL(oauthStateString, oauth2.SetAuthURLParam("response_mode", "form_post")) + "&scope=name email"
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodMicrosoft:
			if oauth.OAuthProviders.MicrosoftConfig == nil {
				log.Debug("Microsoft OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.AuthRecipeMethodMicrosoft)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProviders.MicrosoftConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodMicrosoft
			url := oauth.OAuthProviders.MicrosoftConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodTwitch:
			if oauth.OAuthProviders.TwitchConfig == nil {
				log.Debug("Twitch OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := memorystore.Provider.SetState(oauthStateString, constants.AuthRecipeMethodTwitch)
			if err != nil {
				log.Debug("Error setting state: ", err)
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProviders.TwitchConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodTwitch
			url := oauth.OAuthProviders.TwitchConfig.AuthCodeURL(oauthStateString)
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
