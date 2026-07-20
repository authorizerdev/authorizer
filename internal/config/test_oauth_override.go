package config

import "github.com/authorizerdev/authorizer/internal/constants"

// TestOAuthBaseURL returns the configured e2e-playground mock-IdP base URL
// for the given social-login provider, or "" when unset. These fields are
// only ever populated by e2e-playground/docker-compose.yml; every real
// deployment leaves them empty, so callers fall back to production endpoints.
func (c *Config) TestOAuthBaseURL(provider string) string {
	switch provider {
	case constants.AuthRecipeMethodGoogle:
		return c.TestOAuthGoogleBaseURL
	case constants.AuthRecipeMethodGithub:
		return c.TestOAuthGithubBaseURL
	case constants.AuthRecipeMethodFacebook:
		return c.TestOAuthFacebookBaseURL
	case constants.AuthRecipeMethodLinkedIn:
		return c.TestOAuthLinkedinBaseURL
	case constants.AuthRecipeMethodApple:
		return c.TestOAuthAppleBaseURL
	case constants.AuthRecipeMethodTwitter:
		return c.TestOAuthTwitterBaseURL
	case constants.AuthRecipeMethodDiscord:
		return c.TestOAuthDiscordBaseURL
	case constants.AuthRecipeMethodMicrosoft:
		return c.TestOAuthMicrosoftBaseURL
	case constants.AuthRecipeMethodTwitch:
		return c.TestOAuthTwitchBaseURL
	case constants.AuthRecipeMethodRoblox:
		return c.TestOAuthRobloxBaseURL
	default:
		return ""
	}
}
