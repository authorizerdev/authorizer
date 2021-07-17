package constants

var (
	// Ref: https://github.com/qor/auth/blob/master/providers/google/google.go
	GoogleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	// Ref: https://github.com/qor/auth/blob/master/providers/facebook/facebook.go#L18
	FacebookUserInfoURL = "https://graph.facebook.com/me?access_token="
	// Ref: https://docs.github.com/en/developers/apps/building-github-apps/identifying-and-authorizing-users-for-github-apps#3-your-github-app-accesses-the-api-with-the-users-access-token
	GithubUserInfoURL = "https://api.github.com/user"
)
