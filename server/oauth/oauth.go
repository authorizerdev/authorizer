package oauth

import (
	"github.com/yauthdev/yauth/server/constants"
	"golang.org/x/oauth2"
	facebookOAuth2 "golang.org/x/oauth2/facebook"
	githubOAuth2 "golang.org/x/oauth2/github"
	googleOAuth2 "golang.org/x/oauth2/google"
)

type OAuthProviders struct {
	GoogleConfig   *oauth2.Config
	GithubConfig   *oauth2.Config
	FacebookConfig *oauth2.Config
}

var OAuthProvider OAuthProviders

func init() {
	if constants.GOOGLE_CLIENT_ID != "" && constants.GOOGLE_CLIENT_SECRET != "" {
		OAuthProvider.GoogleConfig = &oauth2.Config{
			ClientID:     constants.GOOGLE_CLIENT_ID,
			ClientSecret: constants.GOOGLE_CLIENT_SECRET,
			RedirectURL:  constants.SERVER_URL + "/callback/google",
			Endpoint:     googleOAuth2.Endpoint,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		}
	}
	if constants.GITHUB_CLIENT_ID != "" && constants.GITHUB_CLIENT_SECRET != "" {
		OAuthProvider.GoogleConfig = &oauth2.Config{
			ClientID:     constants.GITHUB_CLIENT_ID,
			ClientSecret: constants.GITHUB_CLIENT_SECRET,
			RedirectURL:  constants.SERVER_URL + "/callback/github",
			Endpoint:     githubOAuth2.Endpoint,
		}
	}
	if constants.FACEBOOK_CLIENT_ID != "" && constants.FACEBOOK_CLIENT_SECRET != "" {
		OAuthProvider.GoogleConfig = &oauth2.Config{
			ClientID:     constants.FACEBOOK_CLIENT_ID,
			ClientSecret: constants.FACEBOOK_CLIENT_SECRET,
			RedirectURL:  constants.SERVER_URL + "/callback/facebook/",
			Endpoint:     facebookOAuth2.Endpoint,
		}
	}
}
