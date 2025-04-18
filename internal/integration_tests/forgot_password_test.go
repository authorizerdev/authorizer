package integration_tests

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestForgotPassword tests the forgot password functionality
func TestForgotPassword(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "forgot_password_test_" + uuid.New().String() + "@authorizer.dev"
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
	t.Run("should fail for invalid email", func(t *testing.T) {
		forgotPasswordReq := &model.ForgotPasswordInput{
			Email: refs.NewStringRef("invalid-email@gmail.com"),
		}
		forgotPasswordRes, err := ts.GraphQLProvider.ForgotPassword(ctx, forgotPasswordReq)
		assert.Error(t, err)
		assert.Nil(t, forgotPasswordRes)
	})

	t.Run("should send forgot password email", func(t *testing.T) {
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
	})
}
