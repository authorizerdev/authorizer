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
	if envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID) != "" && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret) != "" {
		p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			log.Fatalln("error creating oidc provider for google:", err)
		}
		OIDCProviders.GoogleOIDC = p
		OAuthProviders.GoogleConfig = &oauth2.Config{
			ClientID:     envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID),
			ClientSecret: envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret),
			RedirectURL:  "/oauth_callback/google",
			Endpoint:     OIDCProviders.GoogleOIDC.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}
	}
	if envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID) != "" && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret) != "" {
		OAuthProviders.GithubConfig = &oauth2.Config{
			ClientID:     envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID),
			ClientSecret: envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret),
			RedirectURL:  "/oauth_callback/github",
			Endpoint:     githubOAuth2.Endpoint,
		}
	}
	if envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID) != "" && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID) != "" {
		OAuthProviders.FacebookConfig = &oauth2.Config{
			ClientID:     envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID),
			ClientSecret: envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientSecret),
			RedirectURL:  "/oauth_callback/facebook",
			Endpoint:     facebookOAuth2.Endpoint,
			Scopes:       []string{"public_profile", "email"},
		}
	}
}
