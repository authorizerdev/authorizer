package utils

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	validEmail := "lakhan@gmail.com"
	invalidEmail1 := "lakhan"
	invalidEmail2 := "lakhan.me"

	assert.True(t, IsValidEmail(validEmail), "it should be valid email")
	assert.False(t, IsValidEmail(invalidEmail1), "it should be invalid email")
	assert.False(t, IsValidEmail(invalidEmail2), "it should be invalid email")
}

func TestIsValidOrigin(t *testing.T) {
	// don't use portocal(http/https) for ALLOWED_ORIGINS while testing,
	// as we trim them off while running the main function
	constants.ALLOWED_ORIGINS = []string{"localhost:8080", "*.google.com", "*.google.in", "*abc.*"}

	assert.False(t, IsValidOrigin("http://myapp.com"), "it should be invalid origin")
	assert.False(t, IsValidOrigin("http://appgoogle.com"), "it should be invalid origin")
	assert.True(t, IsValidOrigin("http://app.google.com"), "it should be valid origin")
	assert.False(t, IsValidOrigin("http://app.google.ind"), "it should be invalid origin")
	assert.True(t, IsValidOrigin("http://app.google.in"), "it should be valid origin")
	assert.True(t, IsValidOrigin("http://xyx.abc.com"), "it should be valid origin")
	assert.True(t, IsValidOrigin("http://xyx.abc.in"), "it should be valid origin")
	assert.True(t, IsValidOrigin("http://xyxabc.in"), "it should be valid origin")
	assert.True(t, IsValidOrigin("http://localhost:8080"), "it should be valid origin")
}
