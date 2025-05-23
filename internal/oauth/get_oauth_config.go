package oauth

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
)

// GetOAuthConfig returns the OAuth config for the given provider
func (o *oauthProvider) GetOAuthConfig(ctx *gin.Context, provider string) (*oauth2.Config, error) {
	hostname := parsers.GetHost(ctx)
	redirectURL := fmt.Sprintf("%s/oauth_callback/%s", hostname, provider)
	var clientID, clientSecret string
	var endPoint *oauth2.Endpoint
	var scopes []string
	switch provider {
	case constants.AuthRecipeMethodGoogle:
		if o.GoogleClientID != "" && o.GoogleClientSecret != "" {
			clientID = o.GoogleClientID
			clientSecret = o.GoogleClientSecret
			endPoint = &endpoints.Google
			scopes = o.GoogleScopes
		}
	case constants.AuthRecipeMethodGithub:
		if o.GithubClientID != "" && o.GithubClientSecret != "" {
			clientID = o.GithubClientID
			clientSecret = o.GithubClientSecret
			endPoint = &endpoints.GitHub
			scopes = o.GithubScopes
		}
	case constants.AuthRecipeMethodFacebook:
		if o.FacebookClientID != "" && o.FacebookClientSecret != "" {
			clientID = o.FacebookClientID
			clientSecret = o.FacebookClientSecret
			endPoint = &endpoints.Facebook
			scopes = o.FacebookScopes
		}
	case constants.AuthRecipeMethodLinkedIn:
		if o.LinkedinClientID != "" && o.LinkedinClientSecret != "" {
			clientID = o.LinkedinClientID
			clientSecret = o.LinkedinClientSecret
			endPoint = &endpoints.LinkedIn
			scopes = o.LinkedinScopes
		}
	case constants.AuthRecipeMethodApple:
		if o.AppleClientID != "" && o.AppleClientSecret != "" {
			clientID = o.AppleClientID
			clientSecret = o.AppleClientSecret
			endPoint = &endpoints.Apple
			scopes = o.AppleScopes
		}
	case constants.AuthRecipeMethodTwitter:
		if o.TwitterClientID != "" && o.TwitterClientSecret != "" {
			clientID = o.TwitterClientID
			clientSecret = o.TwitterClientSecret
			endPoint = &endpoints.X
			scopes = o.TwitterScopes
		}
	case constants.AuthRecipeMethodDiscord:
		if o.DiscordClientID != "" && o.DiscordClientSecret != "" {
			clientID = o.DiscordClientID
			clientSecret = o.DiscordClientSecret
			endPoint = &endpoints.Discord
			scopes = o.DiscordScopes
		}
	case constants.AuthRecipeMethodMicrosoft:
		if o.MicrosoftClientID != "" && o.MicrosoftClientSecret != "" {
			ep := endpoints.AzureAD(o.MicrosoftTenantID)
			clientID = o.MicrosoftClientID
			clientSecret = o.MicrosoftClientSecret
			endPoint = &ep
			scopes = o.MicrosoftScopes
		}
	case constants.AuthRecipeMethodTwitch:
		if o.TwitchClientID != "" && o.TwitchClientSecret != "" {
			clientID = o.TwitchClientID
			clientSecret = o.TwitchClientSecret
			endPoint = &endpoints.Twitch
			scopes = o.TwitchScopes
		}
	case constants.AuthRecipeMethodRoblox:
		if o.RobloxClientID != "" && o.RobloxClientSecret != "" {
			clientID = o.RobloxClientID
			clientSecret = o.RobloxClientSecret
			endPoint = &oauth2.Endpoint{
				AuthURL:  "https://apis.roblox.com/oauth/v1/authorize",
				TokenURL: "https://apis.roblox.com/oauth/v1/token",
			}
			scopes = o.RobloxScopes
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("client ID or client secret is empty for provider: %s", provider)
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     *endPoint,
		Scopes:       scopes,
	}, nil
}
