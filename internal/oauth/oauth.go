package oauth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/appengine/log"

	facebookOAuth2 "golang.org/x/oauth2/facebook"
	githubOAuth2 "golang.org/x/oauth2/github"
	linkedInOAuth2 "golang.org/x/oauth2/linkedin"
	microsoftOAuth2 "golang.org/x/oauth2/microsoft"
	twitchOAuth2 "golang.org/x/oauth2/twitch"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memorystore"
)

const (
	microsoftCommonTenant = "common"
)

// OAuthProviders is a struct that contains reference all the OAuth providers
type OAuthProvider struct {
	GoogleConfig    *oauth2.Config
	GithubConfig    *oauth2.Config
	FacebookConfig  *oauth2.Config
	LinkedInConfig  *oauth2.Config
	AppleConfig     *oauth2.Config
	DiscordConfig   *oauth2.Config
	TwitterConfig   *oauth2.Config
	MicrosoftConfig *oauth2.Config
	TwitchConfig    *oauth2.Config
	RobloxConfig    *oauth2.Config
}

// OIDCProviders is a struct that contains reference all the OpenID providers
type OIDCProvider struct {
	GoogleOIDC    *oidc.Provider
	MicrosoftOIDC *oidc.Provider
	TwitchOIDC    *oidc.Provider
}

var (
	// OAuthProviders is a global variable that contains instance for all enabled the OAuth providers
	OAuthProviders OAuthProvider
	// OIDCProviders is a global variable that contains instance for all enabled the OpenID providers
	OIDCProviders OIDCProvider
)

// InitOAuth initializes the OAuth providers based on EnvData
func InitOAuth() error {
	ctx := context.Background()
	googleClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID)
	if err != nil {
		googleClientID = ""
	}
	googleClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret)
	if err != nil {
		googleClientSecret = ""
	}
	if googleClientID != "" && googleClientSecret != "" {
		p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			return err
		}
		OIDCProviders.GoogleOIDC = p
		OAuthProviders.GoogleConfig = &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  "/oauth_callback/google",
			Endpoint:     OIDCProviders.GoogleOIDC.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}
	}

	githubClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID)
	if err != nil {
		githubClientID = ""
	}
	githubClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret)
	if err != nil {
		githubClientSecret = ""
	}
	if githubClientID != "" && githubClientSecret != "" {
		OAuthProviders.GithubConfig = &oauth2.Config{
			ClientID:     githubClientID,
			ClientSecret: githubClientSecret,
			RedirectURL:  "/oauth_callback/github",
			Endpoint:     githubOAuth2.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		}
	}

	facebookClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID)
	if err != nil {
		facebookClientID = ""
	}
	facebookClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientSecret)
	if err != nil {
		facebookClientSecret = ""
	}
	if facebookClientID != "" && facebookClientSecret != "" {
		OAuthProviders.FacebookConfig = &oauth2.Config{
			ClientID:     facebookClientID,
			ClientSecret: facebookClientSecret,
			RedirectURL:  "/oauth_callback/facebook",
			Endpoint:     facebookOAuth2.Endpoint,
			Scopes:       []string{"public_profile", "email"},
		}
	}

	linkedInClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyLinkedInClientID)
	if err != nil {
		linkedInClientID = ""
	}
	linkedInClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyLinkedInClientSecret)
	if err != nil {
		linkedInClientSecret = ""
	}
	if linkedInClientID != "" && linkedInClientSecret != "" {
		OAuthProviders.LinkedInConfig = &oauth2.Config{
			ClientID:     linkedInClientID,
			ClientSecret: linkedInClientSecret,
			RedirectURL:  "/oauth_callback/linkedin",
			Endpoint:     linkedInOAuth2.Endpoint,
			Scopes:       []string{"r_liteprofile", "r_emailaddress"},
		}
	}

	appleClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppleClientID)
	if err != nil {
		appleClientID = ""
	}
	appleClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppleClientSecret)
	if err != nil {
		appleClientSecret = ""
	}
	if appleClientID != "" && appleClientSecret != "" {
		OAuthProviders.AppleConfig = &oauth2.Config{
			ClientID:     appleClientID,
			ClientSecret: appleClientSecret,
			RedirectURL:  "/oauth_callback/apple",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://appleid.apple.com/auth/authorize",
				TokenURL: "https://appleid.apple.com/auth/token",
			},
		}
	}

	discordClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDiscordClientID)
	if err != nil {
		discordClientID = ""
	}
	discordClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDiscordClientSecret)
	if err != nil {
		discordClientSecret = ""
	}
	if discordClientID != "" && discordClientSecret != "" {
		OAuthProviders.DiscordConfig = &oauth2.Config{
			ClientID:     discordClientID,
			ClientSecret: discordClientSecret,
			RedirectURL:  "/oauth_callback/discord",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
			Scopes: []string{"identify", "email"},
		}
	}

	twitterClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwitterClientID)
	if err != nil {
		twitterClientID = ""
	}
	twitterClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwitterClientSecret)
	if err != nil {
		twitterClientSecret = ""
	}
	if twitterClientID != "" && twitterClientSecret != "" {
		OAuthProviders.TwitterConfig = &oauth2.Config{
			ClientID:     twitterClientID,
			ClientSecret: twitterClientSecret,
			RedirectURL:  "/oauth_callback/twitter",
			Endpoint: oauth2.Endpoint{
				// Endpoint is currently not yet part of oauth2-package. See https://go-review.googlesource.com/c/oauth2/+/350889 for status
				AuthURL:   "https://twitter.com/i/oauth2/authorize",
				TokenURL:  "https://api.twitter.com/2/oauth2/token",
				AuthStyle: oauth2.AuthStyleInHeader,
			},
			Scopes: []string{"tweet.read", "users.read"},
		}
	}

	microsoftClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyMicrosoftClientID)
	if err != nil {
		microsoftClientID = ""
	}
	microsoftClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyMicrosoftClientSecret)
	if err != nil {
		microsoftClientSecret = ""
	}
	microsoftActiveDirTenantID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyMicrosoftActiveDirectoryTenantID)
	if err != nil || microsoftActiveDirTenantID == "" {
		microsoftActiveDirTenantID = microsoftCommonTenant
	}
	if microsoftClientID != "" && microsoftClientSecret != "" {
		if microsoftActiveDirTenantID == microsoftCommonTenant {
			ctx = oidc.InsecureIssuerURLContext(ctx, fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", microsoftActiveDirTenantID))
		}
		p, err := oidc.NewProvider(ctx, fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", microsoftActiveDirTenantID))
		if err != nil {
			log.Debugf(ctx, "Error while creating OIDC provider for Microsoft: %v", err)
			return err
		}
		OIDCProviders.MicrosoftOIDC = p
		OAuthProviders.MicrosoftConfig = &oauth2.Config{
			ClientID:     microsoftClientID,
			ClientSecret: microsoftClientSecret,
			RedirectURL:  "/oauth_callback/microsoft",
			Endpoint:     microsoftOAuth2.AzureADEndpoint(microsoftActiveDirTenantID),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}
	}

	twitchClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwitchClientID)
	if err != nil {
		twitchClientID = ""
	}
	twitchClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwitchClientSecret)
	if err != nil {
		twitchClientSecret = ""
	}

	if twitchClientID != "" && twitchClientSecret != "" {
		p, err := oidc.NewProvider(ctx, "https://id.twitch.tv/oauth2")
		if err != nil {
			log.Debugf(ctx, "Error while creating OIDC provider for Twitch: %v", err)
			return err
		}

		OIDCProviders.TwitchOIDC = p
		OAuthProviders.TwitchConfig = &oauth2.Config{
			ClientID:     twitchClientID,
			ClientSecret: twitchClientSecret,
			RedirectURL:  "/oauth_callback/twitch",
			Endpoint:     twitchOAuth2.Endpoint,
			Scopes:       []string{oidc.ScopeOpenID},
		}
	}

	robloxClientID, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyRobloxClientID)
	if err != nil {
		robloxClientID = ""
	}
	robloxClientSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyRobloxClientSecret)
	if err != nil {
		robloxClientSecret = ""
	}
	if robloxClientID != "" && robloxClientSecret != "" {
		OAuthProviders.RobloxConfig = &oauth2.Config{
			ClientID:     robloxClientID,
			ClientSecret: robloxClientSecret,
			RedirectURL:  "/oauth_callback/roblox",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://apis.roblox.com/oauth/v1/authorize",
				TokenURL: "https://apis.roblox.com/oauth/v1/token",
			},
			Scopes: []string{oidc.ScopeOpenID, "profile"},
		}
	}
	return nil
}
