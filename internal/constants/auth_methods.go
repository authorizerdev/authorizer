package constants

const (
	// AuthRecipeMethodBasicAuth is the basic_auth auth method
	AuthRecipeMethodBasicAuth = "basic_auth"
	// AuthRecipeMethodMobileBasicAuth is the mobile basic_auth method, where user can signup using mobile number and password
	AuthRecipeMethodMobileBasicAuth = "mobile_basic_auth"
	// AuthRecipeMethodMagicLinkLogin is the magic_link_login auth method
	AuthRecipeMethodMagicLinkLogin = "magic_link_login"
	// AuthRecipeMethodMobileOTP is the mobile_otp auth method
	AuthRecipeMethodMobileOTP = "mobile_otp"

	// AuthRecipeMethodGoogle is the google auth method
	AuthRecipeMethodGoogle = "google"
	// AuthRecipeMethodGithub is the github auth method
	AuthRecipeMethodGithub = "github"
	// AuthRecipeMethodFacebook is the facebook auth method
	AuthRecipeMethodFacebook = "facebook"
	// AuthRecipeMethodLinkedin is the linkedin auth method
	AuthRecipeMethodLinkedIn = "linkedin"
	// AuthRecipeMethodApple is the apple auth method
	AuthRecipeMethodApple = "apple"
	// AuthRecipeMethodDiscord is the discord auth method
	AuthRecipeMethodDiscord = "discord"
	// AuthRecipeMethodTwitter is the twitter auth method
	AuthRecipeMethodTwitter = "twitter"
	// AuthRecipeMethodMicrosoft is the microsoft auth method
	AuthRecipeMethodMicrosoft = "microsoft"
	// AuthRecipeMethodTwitch is the twitch auth method
	AuthRecipeMethodTwitch = "twitch"
	// AuthRecipeMethodRoblox is the roblox auth method
	AuthRecipeMethodRoblox = "roblox"
	// AuthRecipeMethodSSO is the login_method stamped on sessions established via
	// a per-org OIDC SSO broker flow. It namespaces the memory-store session key
	// ("sso:<user_id>") the same way the social recipes do.
	AuthRecipeMethodSSO = "sso"

	// AuthRecipeMethodServiceAccount is the login_method stamped on machine
	// access tokens issued via the client_credentials grant (RFC 6749 §4.4).
	// It is not a human login recipe — it namespaces the memory-store session
	// key ("service_account:<id>") so ValidateAccessToken derives the same key
	// the token endpoint registered the token under, keeping machine tokens on
	// the existing stateful validation path with zero special-casing.
	AuthRecipeMethodServiceAccount = "service_account"
)
