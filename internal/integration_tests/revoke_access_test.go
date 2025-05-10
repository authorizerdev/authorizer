package integration_tests

import (
	"fmt"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestRevokeAccessUser tests the revoke access functionality by the admin
func TestRevokeAccessUser(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "revoke_access_test_" + uuid.New().String() + "@authorizer.dev"
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
		revokeAccessDets, err := ts.GraphQLProvider.RevokeAccess(ctx, &model.UpdateAccessInput{
			UserID: signupRes.User.ID,
		})
		require.Error(t, err)
		require.Nil(t, revokeAccessDets)
	})

	t.Run("should fail with blank userid", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		_, err = ts.GraphQLProvider.RevokeAccess(ctx, &model.UpdateAccessInput{
			UserID: "",
		})
		require.Error(t, err)
	})

	t.Run("should fail with unknown userid", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		_, err = ts.GraphQLProvider.RevokeAccess(ctx, &model.UpdateAccessInput{
			UserID: uuid.NewString(),
		})
		require.Error(t, err)
	})

	t.Run("should revoke access", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		revokeAccessDets, err := ts.GraphQLProvider.RevokeAccess(ctx, &model.UpdateAccessInput{
			UserID: signupRes.User.ID,
		})
		require.NoError(t, err)
		assert.NotNil(t, revokeAccessDets)
	})
}
