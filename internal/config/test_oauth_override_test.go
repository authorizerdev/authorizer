package config

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/stretchr/testify/assert"
)

// TestTestOAuthBaseURL_ProductionDefault_ReturnsEmpty proves the no-op claim:
// with Env unset (production default), every provider - including a real,
// supported one - returns "", so callers fall through to real endpoints.
func TestTestOAuthBaseURL_ProductionDefault_ReturnsEmpty(t *testing.T) {
	c := &Config{}
	assert.Equal(t, "", c.TestOAuthBaseURL(constants.AuthRecipeMethodGoogle))
	assert.Equal(t, "", c.TestOAuthBaseURL(constants.AuthRecipeMethodDiscord))
}

// TestTestOAuthBaseURL_TestEnv_StillReturnsEmpty proves E2EEnv and TestEnv are
// genuinely distinct: the internal/integration_tests suite runs with
// Env=TestEnv, and must not accidentally start routing through the
// e2e-playground mock.
func TestTestOAuthBaseURL_TestEnv_StillReturnsEmpty(t *testing.T) {
	c := &Config{Env: constants.TestEnv}
	assert.Equal(t, "", c.TestOAuthBaseURL(constants.AuthRecipeMethodGoogle))
}

// TestTestOAuthBaseURL_E2EEnv_ReturnsMockURLPerProvider proves the actual
// escape hatch: under --env=e2e, every one of the 10 supported providers
// resolves to the same mock-oauth host with its own path segment, and an
// unknown/unsupported provider name still returns "".
func TestTestOAuthBaseURL_E2EEnv_ReturnsMockURLPerProvider(t *testing.T) {
	c := &Config{Env: constants.E2EEnv}
	assert.Equal(t, "http://mock-oauth:4000/google", c.TestOAuthBaseURL(constants.AuthRecipeMethodGoogle))
	assert.Equal(t, "http://mock-oauth:4000/github", c.TestOAuthBaseURL(constants.AuthRecipeMethodGithub))
	assert.Equal(t, "http://mock-oauth:4000/roblox", c.TestOAuthBaseURL(constants.AuthRecipeMethodRoblox))
	assert.Equal(t, "", c.TestOAuthBaseURL("not-a-real-provider"), "unknown provider must return empty string")
}
