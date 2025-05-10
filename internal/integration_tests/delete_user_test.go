package integration_tests

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestDeleteUser tests the delete user functionality by the admin
func TestDeleteUser(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "delete_user_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpInput{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	require.NotNil(t, signupRes.User)

	t.Run("should fail without admin cookie", func(t *testing.T) {
		deleteRes, err := ts.GraphQLProvider.DeleteUser(ctx, &model.DeleteUserInput{Email: email})
		require.Error(t, err)
		require.Nil(t, deleteRes)
	})

	t.Run("should delete user", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		deleteRes, err := ts.GraphQLProvider.DeleteUser(ctx, &model.DeleteUserInput{Email: email})
		require.NoError(t, err)
		require.NotNil(t, deleteRes)
	})
}
