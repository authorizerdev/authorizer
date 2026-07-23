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

	// TwitterUserInfoURL requests confirmed_email as a sparse-fieldset field
	// alongside the always-present id/name/profile_image_url/username. Per
	// X's documented sparse fieldset behavior, an unauthorized-or-unrequested
	// field is simply omitted from the response rather than erroring the
	// request, so asking for it is safe for every operator - whether or not
	// they've opted in below.
	//
	// X only populates confirmed_email when BOTH of these are true for the
	// operator's own X Developer App:
	//   1. The OAuth request includes the `users.email` scope - add it via
	//      the --twitter-scopes flag (defaultTwitterScopes in cmd/root.go is
	//      deliberately left unchanged; requesting a scope the operator's
	//      app dashboard hasn't enabled risks X rejecting the whole
	//      authorization request for THEIR app, so this is opt-in only).
	//   2. "Request email from users" is enabled in that app's X Developer
	//      dashboard.
	// Without both, processTwitterUserInfo falls back to the synthetic
	// per-id email, which still correctly prevents duplicate accounts, just
	// without a real deliverable address.
	TwitterUserInfoURL = "https://api.twitter.com/2/users/me?user.fields=confirmed_email,id,name,profile_image_url,username"

	// RobloxUserInfoURL is the URL to get user info from Roblox
	RobloxUserInfoURL = "https://apis.roblox.com/oauth/v1/userinfo"

	DiscordUserInfoURL = "https://discord.com/api/oauth2/@me"
	// Get microsoft user info.
	// Ref: https://learn.microsoft.com/en-us/azure/active-directory/develop/userinfo
	MicrosoftUserInfoURL = "https://graph.microsoft.com/oidc/userinfo"
)
