package oauth

import (
	"context"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	facebookOAuth2 "golang.org/x/oauth2/facebook"
	githubOAuth2 "golang.org/x/oauth2/github"
)

type OAuthProvider struct {
	GoogleConfig   *oauth2.Config
	GithubConfig   *oauth2.Config
	FacebookConfig *oauth2.Config
}

type OIDCProvider struct {
	GoogleOIDC *oidc.Provider
}

var (
	OAuthProviders OAuthProvider
	OIDCProviders  OIDCProvider
)

func InitOAuth() {
	ctx := context.Background()
	if constants.EnvData.GOOGLE_CLIENT_ID != "" && constants.EnvData.GOOGLE_CLIENT_SECRET != "" {
		p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			log.Fatalln("error creating oidc provider for google:", err)
		}
		OIDCProviders.GoogleOIDC = p
		OAuthProviders.GoogleConfig = &oauth2.Config{
			ClientID:     constants.EnvData.GOOGLE_CLIENT_ID,
			ClientSecret: constants.EnvData.GOOGLE_CLIENT_SECRET,
			RedirectURL:  constants.EnvData.AUTHORIZER_URL + "/oauth_callback/google",
			Endpoint:     OIDCProviders.GoogleOIDC.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		}
	}
	if constants.EnvData.GITHUB_CLIENT_ID != "" && constants.EnvData.GITHUB_CLIENT_SECRET != "" {
		OAuthProviders.GithubConfig = &oauth2.Config{
			ClientID:     constants.EnvData.GITHUB_CLIENT_ID,
			ClientSecret: constants.EnvData.GITHUB_CLIENT_SECRET,
			RedirectURL:  constants.EnvData.AUTHORIZER_URL + "/oauth_callback/github",
			Endpoint:     githubOAuth2.Endpoint,
		}
	}
	if constants.EnvData.FACEBOOK_CLIENT_ID != "" && constants.EnvData.FACEBOOK_CLIENT_SECRET != "" {
		OAuthProviders.FacebookConfig = &oauth2.Config{
			ClientID:     constants.EnvData.FACEBOOK_CLIENT_ID,
			ClientSecret: constants.EnvData.FACEBOOK_CLIENT_SECRET,
			RedirectURL:  constants.EnvData.AUTHORIZER_URL + "/oauth_callback/facebook",
			Endpoint:     facebookOAuth2.Endpoint,
			Scopes:       []string{"public_profile", "email"},
		}
	}
}
