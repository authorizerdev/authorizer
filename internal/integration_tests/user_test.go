package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUser tests the _user admin query
func TestUser(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "user_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	require.NotNil(t, signupRes.User)

	t.Run("should fail without admin auth", func(t *testing.T) {
		req.Header.Set("Cookie", "")
		user, err := ts.GraphQLProvider.User(ctx, &model.GetUserRequest{
			ID: refs.NewStringRef(signupRes.User.ID),
		})
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("should get user by ID", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		user, err := ts.GraphQLProvider.User(ctx, &model.GetUserRequest{
			ID: refs.NewStringRef(signupRes.User.ID),
		})
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, signupRes.User.ID, user.ID)
		assert.Equal(t, email, *user.Email)
	})

	t.Run("should get user by email", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		user, err := ts.GraphQLProvider.User(ctx, &model.GetUserRequest{
			Email: refs.NewStringRef(email),
		})
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, *user.Email)
	})

	t.Run("should fail for non-existent user", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		user, err := ts.GraphQLProvider.User(ctx, &model.GetUserRequest{
			ID: refs.NewStringRef(uuid.New().String()),
		})
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}
