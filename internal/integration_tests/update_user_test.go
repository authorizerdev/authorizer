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

	t.Run("should reject duplicate phone number", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		// Create a second user
		email2 := "update_user_phone_" + uuid.New().String() + "@authorizer.dev"
		signupRes2, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email2,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, signupRes2)

		// Assign a phone number to user A
		phoneA := refs.NewStringRef("+1234567890")
		_, err = ts.GraphQLProvider.UpdateUser(ctx, &model.UpdateUserRequest{
			ID:          signupRes.User.ID,
			PhoneNumber: phoneA,
		})
		require.NoError(t, err)

		// Try to assign the same phone number to user B
		_, err = ts.GraphQLProvider.UpdateUser(ctx, &model.UpdateUserRequest{
			ID:          signupRes2.User.ID,
			PhoneNumber: phoneA,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})
}
