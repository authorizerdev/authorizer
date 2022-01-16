package oauth

import (
	"context"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	facebookOAuth2 "golang.org/x/oauth2/facebook"
	githubOAuth2 "golang.org/x/oauth2/github"
)

// OAuthProviders is a struct that contains reference all the OAuth providers
type OAuthProvider struct {
	GoogleConfig   *oauth2.Config
	GithubConfig   *oauth2.Config
	FacebookConfig *oauth2.Config
}

// OIDCProviders is a struct that contains reference all the OpenID providers
type OIDCProvider struct {
	GoogleOIDC *oidc.Provider
}

var (
	// OAuthProviders is a global variable that contains instance for all enabled the OAuth providers
	OAuthProviders OAuthProvider
	// OIDCProviders is a global variable that contains instance for all enabled the OpenID providers
	OIDCProviders OIDCProvider
)

// InitOAuth initializes the OAuth providers based on EnvData
func InitOAuth() {
	ctx := context.Background()
	if envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGoogleClientID).(string) != "" && envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGoogleClientSecret).(string) != "" {
		p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			log.Fatalln("error creating oidc provider for google:", err)
		}
		OIDCProviders.GoogleOIDC = p
		OAuthProviders.GoogleConfig = &oauth2.Config{
			ClientID:     envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGoogleClientID).(string),
			ClientSecret: envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGoogleClientSecret).(string),
			RedirectURL:  envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAuthorizerURL).(string) + "/oauth_callback/google",
			Endpoint:     OIDCProviders.GoogleOIDC.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}
	}
	if envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGithubClientID).(string) != "" && envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGithubClientSecret).(string) != "" {
		OAuthProviders.GithubConfig = &oauth2.Config{
			ClientID:     envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGithubClientID).(string),
			ClientSecret: envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGithubClientSecret).(string),
			RedirectURL:  envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAuthorizerURL).(string) + "/oauth_callback/github",
			Endpoint:     githubOAuth2.Endpoint,
		}
	}
	if envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyFacebookClientID).(string) != "" && envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGoogleClientID).(string) != "" {
		OAuthProviders.FacebookConfig = &oauth2.Config{
			ClientID:     envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyFacebookClientID).(string),
			ClientSecret: envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyFacebookClientSecret).(string),
			RedirectURL:  envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAuthorizerURL).(string) + "/oauth_callback/facebook",
			Endpoint:     facebookOAuth2.Endpoint,
			Scopes:       []string{"public_profile", "email"},
		}
	}
}
