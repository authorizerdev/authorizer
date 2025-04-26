package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestDeactivateAccount tests the account deactivation functionality.
func TestDeactivateAccount(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user and login to get tokens
	email := "deactivate_test_" + uuid.New().String() + "@authorizer.dev"
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
	assert.Equal(t, email, *signupRes.User.Email)
	assert.NotEmpty(t, *signupRes.AccessToken)

	// Login to get fresh tokens
	loginReq := &model.LoginInput{
		Email:    &email,
		Password: password,
	}
	loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
	assert.NoError(t, err)
	assert.NotNil(t, loginRes)
	assert.NotEmpty(t, *loginRes.AccessToken)

	// Test cases
	t.Run("should fail deactivate account without access token", func(t *testing.T) {
		// Clear any existing authorization header
		ts.GinContext.Request.Header.Set("Authorization", "")
		deactivateRes, err := ts.GraphQLProvider.DeactivateAccount(ctx)
		assert.Error(t, err)
		assert.Nil(t, deactivateRes)
	})

	t.Run("should fail deactivate account with invalid access token", func(t *testing.T) {
		// Set an invalid token
		ts.GinContext.Request.Header.Set("Authorization", "Bearer invalid_token")
		deactivateRes, err := ts.GraphQLProvider.DeactivateAccount(ctx)
		assert.Error(t, err)
		assert.Nil(t, deactivateRes)
	})

	t.Run("should deactivate account successfully", func(t *testing.T) {
		// Set the valid token
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*loginRes.AccessToken)
		deactivateRes, err := ts.GraphQLProvider.DeactivateAccount(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, deactivateRes)

		t.Run("should fail login after deactivation", func(t *testing.T) {
			// Attempt to login with the deactivated account
			loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
			assert.Error(t, err)
			assert.Nil(t, loginRes)
		})
	})
}
