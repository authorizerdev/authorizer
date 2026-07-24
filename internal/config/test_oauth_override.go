package config

import "github.com/authorizerdev/authorizer/internal/constants"

// e2eMockOAuthBase is the fixed docker-compose-internal address of the
// e2e-playground mock-IdP server (e2e-playground/mocks/mock-oauth). Only
// ever reachable from inside that specific docker-compose network - never
// resolvable in a real deployment, and only consulted at all when
// Env == E2EEnv (see TestOAuthBaseURL below).
const e2eMockOAuthBase = "http://mock-oauth:4000"

// TestOAuthMockBaseOverride lets a Go unit test (internal/oauth,
// internal/http_handlers) point TestOAuthBaseURL at an ephemeral local
// httptest.Server instead of the fixed e2e-playground address, without a
// CLI flag or Config field. Set it, use it, then clear it (t.Cleanup) in the
// same test - it is never read or written by production code, and Env must
// still be E2EEnv for it to take effect at all. Unlike the real
// e2e-playground path, the override replaces the whole computed URL
// verbatim (no provider-path suffix), matching how each unit test's own
// isolated mock server registers its routes at its own base, not under a
// shared /<provider> prefix.
var TestOAuthMockBaseOverride string

// TestOAuthBaseURL returns the e2e-playground mock-IdP base URL for the given
// social-login provider when running under --env=e2e, or "" otherwise. Real
// deployments never set --env=e2e, so callers fall back to production
// endpoints unconditionally - this is a pure function of Config.Env, with no
// separate per-provider flag to configure or forget.
func (c *Config) TestOAuthBaseURL(provider string) string {
	if c.Env != constants.E2EEnv {
		return ""
	}
	if TestOAuthMockBaseOverride != "" {
		return TestOAuthMockBaseOverride
	}
	switch provider {
	case constants.AuthRecipeMethodGoogle,
		constants.AuthRecipeMethodGithub,
		constants.AuthRecipeMethodFacebook,
		constants.AuthRecipeMethodLinkedIn,
		constants.AuthRecipeMethodApple,
		constants.AuthRecipeMethodTwitter,
		constants.AuthRecipeMethodDiscord,
		constants.AuthRecipeMethodMicrosoft,
		constants.AuthRecipeMethodTwitch,
		constants.AuthRecipeMethodRoblox:
		return e2eMockOAuthBase + "/" + provider
	default:
		return ""
	}
}
