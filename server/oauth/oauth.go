package oauth

import (
	"context"

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
func InitOAuth() error {
	ctx := context.Background()
	if envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID) != "" && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret) != "" {
		p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			return err
		}
		OIDCProviders.GoogleOIDC = p
		OAuthProviders.GoogleConfig = &oauth2.Config{
			ClientID:     envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID),
			ClientSecret: envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret),
			RedirectURL:  "/oauth_callback/google",
			Endpoint:     OIDCProviders.GoogleOIDC.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}
	}
	if envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID) != "" && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret) != "" {
		OAuthProviders.GithubConfig = &oauth2.Config{
			ClientID:     envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID),
			ClientSecret: envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret),
			RedirectURL:  "/oauth_callback/github",
			Endpoint:     githubOAuth2.Endpoint,
		}
	}
	if envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID) != "" && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID) != "" {
		OAuthProviders.FacebookConfig = &oauth2.Config{
			ClientID:     envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID),
			ClientSecret: envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientSecret),
			RedirectURL:  "/oauth_callback/facebook",
			Endpoint:     facebookOAuth2.Endpoint,
			Scopes:       []string{"public_profile", "email"},
		}
	}

	return nil
}
