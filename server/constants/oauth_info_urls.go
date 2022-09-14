package constants

const (
	// Ref: https://github.com/qor/auth/blob/master/providers/google/google.go
	// deprecated and not used. instead we follow open id approach for google login
	GoogleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"
	// Ref: https://github.com/qor/auth/blob/master/providers/facebook/facebook.go#L18
	FacebookUserInfoURL = "https://graph.facebook.com/me?fields=id,first_name,last_name,name,email,picture&access_token="
	// Ref: https://docs.github.com/en/developers/apps/building-github-apps/identifying-and-authorizing-users-for-github-apps#3-your-github-app-accesses-the-api-with-the-users-access-token
	GithubUserInfoURL = "https://api.github.com/user"
	// Get github user emails when user info email is empty Ref: https://stackoverflow.com/a/35387123
	GithubUserEmails = "https://api.github.com/user/emails"

	// Ref: https://docs.microsoft.com/en-us/linkedin/shared/integrations/people/profile-api
	LinkedInUserInfoURL = "https://api.linkedin.com/v2/me?projection=(id,localizedFirstName,localizedLastName,emailAddress,profilePicture(displayImage~:playableStreams))"
	LinkedInEmailURL    = "https://api.linkedin.com/v2/emailAddress?q=members&projection=(elements*(handle~))"

	TwitterUserInfoURL = "https://api.twitter.com/2/users/me?user.fields=id,name,profile_image_url,username"
)
