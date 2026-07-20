package config

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestTestOAuthBaseURL(t *testing.T) {
	c := &Config{TestOAuthGoogleBaseURL: "http://mock-oauth:4000/google"}

	assert.Equal(t, "http://mock-oauth:4000/google", c.TestOAuthBaseURL(constants.AuthRecipeMethodGoogle))
	assert.Equal(t, "", c.TestOAuthBaseURL(constants.AuthRecipeMethodGithub), "unset provider must return empty string")
	assert.Equal(t, "", c.TestOAuthBaseURL("not-a-real-provider"), "unknown provider must return empty string")
}
