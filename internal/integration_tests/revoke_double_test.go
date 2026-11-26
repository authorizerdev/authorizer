package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
)

// TestRevoke_Twice: revoking an already-revoked refresh token is the caller's
// state, not a server fault. The memory-store miss must not surface as a
// masked internal error (issue #356); it gets the same typed NotFound as an
// empty session entry.
func TestRevoke_Twice(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "revoke_twice_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
		Scope:    []string{"offline_access"},
	})
	require.NoError(t, err)
	require.NotNil(t, loginRes.RefreshToken)

	req := &model.OAuthRevokeRequest{RefreshToken: *loginRes.RefreshToken}
	_, err = ts.GraphQLProvider.Revoke(ctx, req)
	require.NoError(t, err)

	res, err := ts.GraphQLProvider.Revoke(ctx, req)
	require.Error(t, err)
	assert.Nil(t, res)
	var se *service.Error
	require.ErrorAs(t, err, &se, "a double revoke must be a typed error, not a masked internal one")
	assert.Equal(t, service.KindNotFound, se.Kind)
	_, _, masked := service.MaskInternalError(err)
	assert.False(t, masked)
}
