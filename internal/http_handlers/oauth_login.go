package http_handlers

import (
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// OAuthLoginHandler set host in the oauth state that is useful for redirecting to oauth_callback
func (h *httpProvider) OAuthLoginHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "OAuthLoginHandler").Logger()
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
			log.Debug().Msg("redirect uri is missing")
			c.JSON(400, gin.H{
				"error": "invalid redirect uri",
			})
			return
		}

		if state == "" {
			log.Debug().Msg("state is missing, creating new state")
			state = uuid.New().String()
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
			allowedRoles := h.Config.Roles
			protectedRoles := h.Config.ProtectedRoles
			if !validators.IsValidRoles(rolesSplit, append([]string{}, append(allowedRoles, protectedRoles...)...)) {
				log.Debug().Msg("invalid role")
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
		} else {
			roles = strings.Join(h.Config.DefaultRoles, ",")
		}

		oauthStateString := state + "___" + redirectURI + "___" + roles + "___" + strings.Join(scope, " ")

		provider := c.Param("oauth_provider")
		isProviderConfigured := true
		log := log.With().Str("provider", provider).Logger()

		switch provider {
		case constants.AuthRecipeMethodGoogle:
			if oauth.OAuthProviders.GoogleConfig == nil {
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodGoogle)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
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
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodGithub)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
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
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodFacebook)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
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
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodLinkedIn)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
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
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}

			verifier, challenge := utils.GenerateCodeChallenge()

			err := h.MemoryStoreProvider.SetState(oauthStateString, verifier)
			if err != nil {
				log.Debug().Msg("OAuth provider is not configured")
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.TwitterConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodTwitter
			url := oauth.OAuthProviders.TwitterConfig.AuthCodeURL(oauthStateString, oauth2.SetAuthURLParam("code_challenge", challenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
			c.Redirect(http.StatusTemporaryRedirect, url)

		case constants.AuthRecipeMethodDiscord:
			if oauth.OAuthProviders.DiscordConfig == nil {
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodDiscord)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			oauth.OAuthProviders.DiscordConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodDiscord
			url := oauth.OAuthProviders.DiscordConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodApple:
			if oauth.OAuthProviders.AppleConfig == nil {
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodApple)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
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
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodMicrosoft)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
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
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodTwitch)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProviders.TwitchConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodTwitch
			url := oauth.OAuthProviders.TwitchConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		case constants.AuthRecipeMethodRoblox:
			if oauth.OAuthProviders.RobloxConfig == nil {
				log.Debug().Msg("OAuth provider is not configured")
				isProviderConfigured = false
				break
			}
			err := h.MemoryStoreProvider.SetState(oauthStateString, constants.AuthRecipeMethodRoblox)
			if err != nil {
				log.Debug().Err(err).Msg("Error setting state")
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			// during the init of OAuthProvider authorizer url might be empty
			oauth.OAuthProviders.RobloxConfig.RedirectURL = hostname + "/oauth_callback/" + constants.AuthRecipeMethodRoblox
			url := oauth.OAuthProviders.RobloxConfig.AuthCodeURL(oauthStateString)
			c.Redirect(http.StatusTemporaryRedirect, url)
		default:
			log.Debug().Msg("Invalid OAuth provider")
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
