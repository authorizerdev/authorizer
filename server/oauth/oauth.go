package oauth

import (
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"golang.org/x/oauth2"
	githubOAuth2 "golang.org/x/oauth2/github"
	googleOAuth2 "golang.org/x/oauth2/google"
)

type OAuthProviders struct {
	GoogleConfig *oauth2.Config
	GithubConfig *oauth2.Config
	// FacebookConfig *oauth2.Config
}

var OAuthProvider OAuthProviders

func InitOAuth() {
	log.Println("---> initializing auth")
	if constants.GOOGLE_CLIENT_ID != "" && constants.GOOGLE_CLIENT_SECRET != "" {
		log.Println("---> initializing google auth")
		OAuthProvider.GoogleConfig = &oauth2.Config{
			ClientID:     constants.GOOGLE_CLIENT_ID,
			ClientSecret: constants.GOOGLE_CLIENT_SECRET,
			RedirectURL:  constants.AUTHORIZER_DOMAIN + "/callback/google",
			Endpoint:     googleOAuth2.Endpoint,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		}
	}
	if constants.GITHUB_CLIENT_ID != "" && constants.GITHUB_CLIENT_SECRET != "" {
		log.Println("---> initializing github auth")
		OAuthProvider.GithubConfig = &oauth2.Config{
			ClientID:     constants.GITHUB_CLIENT_ID,
			ClientSecret: constants.GITHUB_CLIENT_SECRET,
			RedirectURL:  constants.AUTHORIZER_DOMAIN + "/callback/github",
			Endpoint:     githubOAuth2.Endpoint,
		}
	}
	// if constants.FACEBOOK_CLIENT_ID != "" && constants.FACEBOOK_CLIENT_SECRET != "" {
	// 	OAuthProvider.FacebookConfig = &oauth2.Config{
	// 		ClientID:     constants.FACEBOOK_CLIENT_ID,
	// 		ClientSecret: constants.FACEBOOK_CLIENT_SECRET,
	// 		RedirectURL: "/callback/facebook/",
	// 		Endpoint:     facebookOAuth2.Endpoint,
	// 	}
	// }
}
