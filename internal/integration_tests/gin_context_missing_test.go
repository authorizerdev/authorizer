package integration_tests

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestGinContextMissing verifies that every GraphQL resolver returns the actual
// GinContextFromContext error immediately when called outside the normal
// Gin HTTP middleware chain (e.g. bare context.Background()).
//
// This can happen if the resolver is invoked from a code path that does not go
// through the HTTP handler that embeds gin.Context with utils.ContextWithGin.
// The previous behaviour was to silently pass nil to service.MetaFromGin, which
// either produced empty RequestMetadata (degraded auth) or panicked inside
// service.ApplyToGin (nil dereference).
//
// gRPC handlers use transport.MetaFromGRPC (already tested in
// internal/grpcsrv/transport/grpc_metadata_test.go — see
// TestMetaFromGRPC_NoMetadata) which degrades gracefully to empty metadata when
// no gRPC incoming-context metadata is present. That is the correct behaviour
// for the gRPC transport and requires no change here.
//
// HTTP (REST) handlers receive gin.Context directly as a function parameter and
// never need to extract it from the context.Context, so they are unaffected.
func TestGinContextMissing(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	// Bare context — no gin.Context embedded.
	ctx := context.Background()
	wantErrSubstr := "could not retrieve gin.Context"

	email := "gin_ctx_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	t.Run("SignUp", func(t *testing.T) {
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})

	t.Run("Login", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &email,
			Password: password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})

	t.Run("Meta", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Meta(ctx)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})

	t.Run("Profile", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Profile(ctx)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})

	t.Run("Logout", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Logout(ctx)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})

	t.Run("Session", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Session(ctx, &model.SessionQueryRequest{})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})

	t.Run("ValidateSession", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})

	t.Run("ForgotPassword", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{Email: &email})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})

	t.Run("AdminLogin", func(t *testing.T) {
		res, err := ts.GraphQLProvider.AdminLogin(ctx, &model.AdminLoginRequest{AdminSecret: "test"})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), wantErrSubstr)
	})
}
