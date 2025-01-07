package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func verifyEmailTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should verify email`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "verify_email." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		user := *res.User
		assert.Equal(t, email, refs.StringValue(user.Email))
		assert.Nil(t, res.AccessToken, "access token should be nil")
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.Nil(t, err)
		assert.Equal(t, email, verificationRequest.Email)

		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.Nil(t, err)
		assert.NotEqual(t, verifyRes.AccessToken, "", "access token should not be empty")
		// Check if phone number is verified
		user1, err := db.Provider.GetUserByEmail(ctx, email)
		assert.NoError(t, err)
		assert.NotNil(t, user1)
		assert.NotNil(t, user1.EmailVerifiedAt)
		cleanData(email)
	})
}
