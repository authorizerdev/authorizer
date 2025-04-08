package integration_tests

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestResendVerifyEmail tests the resend verify email functionality
func TestResendVerifyEmail(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.DisableEmailVerification = false
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "resend_verify_email_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpInput{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	assert.NoError(t, err)
	assert.NotNil(t, signupRes)
	// Expect the user to be nil, as the email is not verified yet
	assert.Nil(t, signupRes.User)

	t.Run("should fail for invalid token", func(t *testing.T) {
		verificationReq := &model.VerifyEmailInput{
			Token: "invalid-token",
		}
		verificationRes, err := ts.GraphQLProvider.VerifyEmail(ctx, verificationReq)
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})

	t.Run("should resend verify email", func(t *testing.T) {
		// Get the verification token from db
		request, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.NotEmpty(t, request.Token)

		// Verify email with an invalid token
		verificationReq := &model.ResendVerifyEmailInput{
			Email:      email,
			Identifier: constants.VerificationTypeBasicAuthSignup,
		}

		res, err := ts.GraphQLProvider.ResendVerifyEmail(ctx, verificationReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Check if the verification request has different token
		request2, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, request2)
		assert.NotEmpty(t, request2.Token)
		assert.NotEqual(t, request.Token, request2.Token)

		// Verify email with the new token
		verificationReq2 := &model.VerifyEmailInput{
			Token: request2.Token,
		}
		verificationRes, err := ts.GraphQLProvider.VerifyEmail(ctx, verificationReq2)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRes)
		assert.NotNil(t, verificationRes.User)
		assert.Equal(t, email, *verificationRes.User.Email)
		assert.Equal(t, true, verificationRes.User.EmailVerified)
	})
}
