package integration_tests

import (
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("should verify email and use correct login method for basic auth", func(t *testing.T) {
		basicAuthEmail := "verify_email_basic_" + uuid.New().String() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &basicAuthEmail,
			Password:        password,
			ConfirmPassword: password,
		})
		assert.NoError(t, err)

		vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, basicAuthEmail, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, vreq)
		// Identifier should be basic_auth_signup, not magic_link_login
		assert.Equal(t, constants.VerificationTypeBasicAuthSignup, vreq.Identifier)

		verifyRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{
			Token: vreq.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, verifyRes)
		assert.NotEmpty(t, verifyRes.AccessToken)
		assert.NotNil(t, verifyRes.User)
		assert.True(t, verifyRes.User.EmailVerified)
	})
	t.Run("should fail for revoked user", func(t *testing.T) {
		revokedEmail := "verify_email_revoked_" + uuid.New().String() + "@authorizer.dev"
		revokedSignupReq := &model.SignUpRequest{
			Email:           &revokedEmail,
			Password:        password,
			ConfirmPassword: password,
		}
		_, err := ts.GraphQLProvider.SignUp(ctx, revokedSignupReq)
		require.NoError(t, err)

		// Get verification token
		vreq, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, revokedEmail, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		require.NotNil(t, vreq)

		// Revoke the user
		user, err := ts.StorageProvider.GetUserByEmail(ctx, revokedEmail)
		require.NoError(t, err)
		now := time.Now().Unix()
		user.RevokedTimestamp = &now
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		// Try to verify email - should fail
		verificationRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{
			Token: vreq.Token,
		})
		assert.Error(t, err)
		assert.Nil(t, verificationRes)
		assert.Contains(t, err.Error(), "revoked")
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
