package integration_tests

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestVerifyEmail tests the verify email functionality
// using the GraphQL API.
// It creates a user, verifies the email, and checks
// the changes in the database.
func TestVerifyEmail(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "verify_email_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpRequest{
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
		verificationReq := &model.VerifyEmailRequest{
			Token: "invalid-token",
		}
		verificationRes, err := ts.GraphQLProvider.VerifyEmail(ctx, verificationReq)
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
	})

	t.Run("should verify email", func(t *testing.T) {
		// Get the verification token from db
		request, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, request)
		assert.NotEmpty(t, request.Token)

		// Verify email with an invalid token
		verificationReq := &model.VerifyEmailRequest{
			Token: request.Token,
		}

		verificationRes, err := ts.GraphQLProvider.VerifyEmail(ctx, verificationReq)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRes)
		assert.NotEmpty(t, verificationRes.AccessToken)
	})
}
