package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/service"
)

// TestGRPCCheckPermissionsFailClosed exercises AuthorizerService.CheckPermissions
// over the in-process bufconn channel when no FGA engine is configured (the
// default test setup wires no AuthzEngine). With a valid bearer token the auth
// interceptor admits the call; the service fails closed with
// service.ErrFgaNotEnabled, mapped to codes.FailedPrecondition with the verbatim
// "fine-grained authorization is not enabled" message.
func TestGRPCCheckPermissionsFailClosed(t *testing.T) {
	conn, _ := bootGRPCBufconn(t)
	c := authorizerv1.NewAuthorizerServiceClient(conn)
	authCtx := surfaceAuthCtx(t, c)

	_, err := c.CheckPermissions(authCtx, &authorizerv1.CheckPermissionsRequest{
		Checks: []*authorizerv1.PermissionCheckInput{
			{Relation: "can_view", Object: "document:1"},
		},
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status error")
	assert.Equal(t, codes.FailedPrecondition, st.Code(),
		"ErrFgaNotEnabled is a typed FailedPrecondition service error")
	assert.Equal(t, service.ErrFgaNotEnabled.Error(), st.Message(),
		"the FGA-disabled message must surface verbatim")
}

// TestGRPCListPermissionsFailClosed mirrors the CheckPermissions fail-closed
// behavior for ListPermissions: with no FGA engine the engine-nil guard denies
// with service.ErrFgaNotEnabled -> codes.FailedPrecondition.
func TestGRPCListPermissionsFailClosed(t *testing.T) {
	conn, _ := bootGRPCBufconn(t)
	c := authorizerv1.NewAuthorizerServiceClient(conn)
	authCtx := surfaceAuthCtx(t, c)

	_, err := c.ListPermissions(authCtx, &authorizerv1.ListPermissionsRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status error")
	assert.Equal(t, codes.FailedPrecondition, st.Code(),
		"ErrFgaNotEnabled is a typed FailedPrecondition service error")
	assert.Equal(t, service.ErrFgaNotEnabled.Error(), st.Message())
}

// TestGRPCCheckPermissionsEmptyChecksRejected verifies the protovalidate
// interceptor enforces CheckPermissionsRequest.checks min_items=1. The caller
// must be authenticated first (auth interceptor runs before validate); an empty
// checks list is then rejected with codes.InvalidArgument before the handler runs.
func TestGRPCCheckPermissionsEmptyChecksRejected(t *testing.T) {
	conn, _ := bootGRPCBufconn(t)
	c := authorizerv1.NewAuthorizerServiceClient(conn)
	authCtx := surfaceAuthCtx(t, c)

	_, err := c.CheckPermissions(authCtx, &authorizerv1.CheckPermissionsRequest{
		Checks: nil, // empty -> violates min_items: 1
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code(),
		"empty checks must be rejected by the protovalidate interceptor, not the handler")
}
