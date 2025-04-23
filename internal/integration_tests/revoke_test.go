package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestRevoke tests the revoke functionality
func TestRevoke(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "revoke_test_" + uuid.New().String() + "@authorizer.dev"
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
	t.Run("should fail for invalid session", func(t *testing.T) {
		revokeReq := &model.OAuthRevokeInput{
			RefreshToken: "invalid_token",
		}
		revokeRes, err := ts.GraphQLProvider.Revoke(ctx, revokeReq)
		assert.Error(t, err)
		assert.Nil(t, revokeRes)
	})

	t.Run("should revoke", func(t *testing.T) {
		// Login request
		loginReq := &model.LoginInput{
			Email:    &email,
			Password: password,
			Scope:    []string{"offline_access"},
		}
		loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
		assert.NotEmpty(t, loginRes.RefreshToken)
		assert.NotEmpty(t, loginRes.AccessToken)

		// Revoke refresh token
		revokeReq := &model.OAuthRevokeInput{
			RefreshToken: *loginRes.RefreshToken,
		}

		revokeRes, err := ts.GraphQLProvider.Revoke(ctx, revokeReq)
		require.NoError(t, err)
		assert.NotNil(t, revokeRes)
		assert.NotEmpty(t, revokeRes.Message)
	})
}
