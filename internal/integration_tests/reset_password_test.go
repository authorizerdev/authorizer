package integration_tests

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestResetPassword tests the reset password functionality
func TestResetPassword(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "reset_password_test_" + uuid.New().String() + "@authorizer.dev"
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
	assert.NotNil(t, signupRes.User)

	// Create forgot password request
	t.Run("should fail for invalid request", func(t *testing.T) {
		resetPasswordReq := &model.ResetPasswordInput{
			Token:           refs.NewStringRef("test"),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		}
		forgotPasswordRes, err := ts.GraphQLProvider.ResetPassword(ctx, resetPasswordReq)
		assert.Error(t, err)
		assert.Nil(t, forgotPasswordRes)
	})

	t.Run("should reset password with verification token", func(t *testing.T) {
		forgotPasswordReq := &model.ForgotPasswordInput{
			Email: refs.NewStringRef(email),
		}
		forgotPasswordRes, err := ts.GraphQLProvider.ForgotPassword(ctx, forgotPasswordReq)
		assert.NoError(t, err)
		assert.NotNil(t, forgotPasswordRes)
		assert.NotEmpty(t, forgotPasswordRes.Message)

		// Validate if the entry is created in db
		request, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.NotEmpty(t, request.Token)
		assert.Equal(t, email, request.Email)

		// Reset password using the token
		resetPasswordReq := &model.ResetPasswordInput{
			Token:           refs.NewStringRef(request.Token),
			Password:        "NewPassword@123",
			ConfirmPassword: "NewPassword@123",
		}

		resetPasswordRes, err := ts.GraphQLProvider.ResetPassword(ctx, resetPasswordReq)
		assert.NoError(t, err)
		assert.NotNil(t, resetPasswordRes)
		assert.NotEmpty(t, resetPasswordRes.Message)

		// Validate if the password is updated in db by logging in
		loginReq := &model.LoginInput{
			Email:    &email,
			Password: "NewPassword@123",
		}
		loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.NotNil(t, loginRes.AccessToken)
	})
}
