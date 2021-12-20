package utils

import (
	"testing"

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
