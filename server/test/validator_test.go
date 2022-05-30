package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	validEmail := "lakhan@gmail.com"
	invalidEmail1 := "lakhan"
	invalidEmail2 := "lakhan.me"

	assert.True(t, validators.IsValidEmail(validEmail), "it should be valid email")
	assert.False(t, validators.IsValidEmail(invalidEmail1), "it should be invalid email")
	assert.False(t, validators.IsValidEmail(invalidEmail2), "it should be invalid email")
}

func TestIsValidOrigin(t *testing.T) {
	// don't use portocal(http/https) for ALLOWED_ORIGINS while testing,
	// as we trim them off while running the main function
	memorystore.Provider.UpdateEnvVariable(constants.SliceStoreIdentifier, constants.EnvKeyAllowedOrigins, []string{"localhost:8080", "*.google.com", "*.google.in", "*abc.*"})
	assert.False(t, validators.IsValidOrigin("http://myapp.com"), "it should be invalid origin")
	assert.False(t, validators.IsValidOrigin("http://appgoogle.com"), "it should be invalid origin")
	assert.True(t, validators.IsValidOrigin("http://app.google.com"), "it should be valid origin")
	assert.False(t, validators.IsValidOrigin("http://app.google.ind"), "it should be invalid origin")
	assert.True(t, validators.IsValidOrigin("http://app.google.in"), "it should be valid origin")
	assert.True(t, validators.IsValidOrigin("http://xyx.abc.com"), "it should be valid origin")
	assert.True(t, validators.IsValidOrigin("http://xyx.abc.in"), "it should be valid origin")
	assert.True(t, validators.IsValidOrigin("http://xyxabc.in"), "it should be valid origin")
	assert.True(t, validators.IsValidOrigin("http://localhost:8080"), "it should be valid origin")
	memorystore.Provider.UpdateEnvVariable(constants.SliceStoreIdentifier, constants.EnvKeyAllowedOrigins, []string{"*"})
}

func TestIsValidIdentifier(t *testing.T) {
	assert.False(t, utils.IsValidVerificationIdentifier("test"), "it should be invalid identifier")
	assert.True(t, utils.IsValidVerificationIdentifier(constants.VerificationTypeBasicAuthSignup), "it should be valid identifier")
	assert.True(t, utils.IsValidVerificationIdentifier(constants.VerificationTypeUpdateEmail), "it should be valid identifier")
	assert.True(t, utils.IsValidVerificationIdentifier(constants.VerificationTypeForgotPassword), "it should be valid identifier")
}

func TestIsValidPassword(t *testing.T) {
	assert.False(t, utils.IsValidPassword("test"), "it should be invalid password")
	assert.False(t, utils.IsValidPassword("Te@1"), "it should be invalid password")
	assert.False(t, utils.IsValidPassword("n*rp7GGTd29V{xx%{pDb@7n{](SD.!+.Mp#*$EHDGk&$pAMf7e#432Sg,Gr](j3n]jV/3F8BJJT+9u9{q=8zK:8u!rpQBaXJp%A+7r!jQj)M(vC$UX,h;;WKm$U6i#7dBnC&2ryKzKd+(y&=Ud)hErT/j;v3t..CM).8nS)9qLtV7pmP;@2QuzDyGfL7KB()k:BpjAGL@bxD%r5gcBfh7$&wutk!wzMfPFY#nkjjqyZbEHku,{jc;gvbYq2)3w=KExnYz9Vbv:;*;?f##faxkULdMpmm&yEfePixzx+[{[38zGN;3TzF;6M#Xy_tMtx:yK*n$bc(bPyGz%EYkC&]ttUF@#aZ%$QZ:u!icF@+"), "it should be invalid password")
	assert.False(t, utils.IsValidPassword("test@123"), "it should be invalid password")
	assert.True(t, utils.IsValidPassword("Test@123"), "it should be valid password")
}
