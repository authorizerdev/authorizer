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

// TestEnableAccessUser tests the enable access functionality by the admin
func TestEnableAccessUser(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "enable_access_test_" + uuid.New().String() + "@authorizer.dev"
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

	t.Run("should fail without admin cookie", func(t *testing.T) {
		revokedUserDets, err := ts.GraphQLProvider.EnableAccess(ctx, &model.UpdateAccessRequest{
			UserID: signupRes.User.ID,
		})
		require.Error(t, err)
		require.Nil(t, revokedUserDets)
	})

	t.Run("should fail with blank userid", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		_, err = ts.GraphQLProvider.EnableAccess(ctx, &model.UpdateAccessRequest{
			UserID: "",
		})
		require.Error(t, err)
	})

	t.Run("should fail with unknown userid", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		_, err = ts.GraphQLProvider.EnableAccess(ctx, &model.UpdateAccessRequest{
			UserID: uuid.NewString(),
		})
		require.Error(t, err)
	})

	t.Run("should enable access user", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		enableAccessDets, err := ts.GraphQLProvider.EnableAccess(ctx, &model.UpdateAccessRequest{
			UserID: signupRes.User.ID,
		})
		require.NoError(t, err)
		assert.NotNil(t, enableAccessDets)
	})
}
