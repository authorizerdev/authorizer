package oauth

import (
	"context"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	githubOAuth2 "golang.org/x/oauth2/github"
)

type OAuthProvider struct {
	GoogleConfig   *oauth2.Config
	GithubConfig   *oauth2.Config
	FacebookConfig *oauth2.Config
}

type OIDCProvider struct {
	GoogleOIDC   *oidc.Provider
	FacebookOIDC *oidc.Provider
}

var (
	OAuthProviders OAuthProvider
	OIDCProviders  OIDCProvider
)

func InitOAuth() {
	ctx := context.Background()
	if constants.GOOGLE_CLIENT_ID != "" && constants.GOOGLE_CLIENT_SECRET != "" {
		provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			log.Fatalln("error configuring Google OpenID provider:", err.Error())
		}

		OIDCProviders.GoogleOIDC = provider
		OAuthProviders.GoogleConfig = &oauth2.Config{
			ClientID:     constants.GOOGLE_CLIENT_ID,
			ClientSecret: constants.GOOGLE_CLIENT_SECRET,
			RedirectURL:  constants.AUTHORIZER_URL + "/oauth_callback/google",
			Endpoint:     OIDCProviders.GoogleOIDC.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		}
	}

	// Github doesn't support OpenID
	// https://stackoverflow.com/questions/52157568/what-is-github-well-known-openid-configuration-url/52164558

	// https://fusionauth.io/docs/v1/tech/identity-providers/openid-connect/github/

	if constants.GITHUB_CLIENT_ID != "" && constants.GITHUB_CLIENT_SECRET != "" {
		OAuthProviders.GithubConfig = &oauth2.Config{
			ClientID:     constants.GITHUB_CLIENT_ID,
			ClientSecret: constants.GITHUB_CLIENT_SECRET,
			RedirectURL:  constants.AUTHORIZER_URL + "/oauth_callback/github",
			Endpoint:     githubOAuth2.Endpoint,
		}
	}

	if constants.FACEBOOK_CLIENT_ID != "" && constants.FACEBOOK_CLIENT_SECRET != "" {
		provider, err := oidc.NewProvider(ctx, "https://www.facebook.com")
		if err != nil {
			log.Fatalln("error configuring Facebook OpenID provider:", err.Error())
		}

		OIDCProviders.FacebookOIDC = provider
		OAuthProviders.FacebookConfig = &oauth2.Config{
			ClientID:     constants.FACEBOOK_CLIENT_ID,
			ClientSecret: constants.FACEBOOK_CLIENT_SECRET,
			RedirectURL:  constants.AUTHORIZER_URL + "/oauth_callback/facebook",
			Endpoint:     OIDCProviders.FacebookOIDC.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "public_profile"},
		}
	}
}
