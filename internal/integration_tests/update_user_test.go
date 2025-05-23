package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateUser tests the update user functionality by the admin
func TestUpdateUser(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "update_user_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	require.NotNil(t, signupRes.User)

	userFirstName := "UpdatedFirstName"
	updateReq := &model.UpdateUserRequest{
		ID:        signupRes.User.ID,
		GivenName: &userFirstName,
	}
	t.Run("should fail without admin cookie", func(t *testing.T) {
		updateRes, err := ts.GraphQLProvider.UpdateUser(ctx, updateReq)
		require.Error(t, err)
		require.Nil(t, updateRes)
	})

	t.Run("should update user", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		updateRes, err := ts.GraphQLProvider.UpdateUser(ctx, updateReq)
		require.NoError(t, err)
		require.NotNil(t, updateRes)
		require.Equal(t, updateRes.ID, signupRes.User.ID)
		require.Equal(t, userFirstName, *updateRes.GivenName)
	})
}
