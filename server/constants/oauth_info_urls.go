package constants

var (
	// Ref: https://github.com/qor/auth/blob/master/providers/google/google.go
	// deprecated and not used. instead we follow open id approach for google login
	GoogleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	// Ref: https://github.com/qor/auth/blob/master/providers/facebook/facebook.go#L18
	FacebookUserInfoURL = "https://graph.facebook.com/me?fields=id,first_name,last_name,name,email,picture&access_token="
	// Ref: https://docs.github.com/en/developers/apps/building-github-apps/identifying-and-authorizing-users-for-github-apps#3-your-github-app-accesses-the-api-with-the-users-access-token
	GithubUserInfoURL = "https://api.github.com/user"
)
