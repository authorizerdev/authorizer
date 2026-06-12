package integration_tests

import (
	"context"
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
// default test setup wires no AuthzEngine). The service fails closed with
// service.ErrFgaNotEnabled before any subject/auth resolution; that plain error
// flows through the ErrorMap interceptor as codes.FailedPrecondition, carrying the
// verbatim "fine-grained authorization is not enabled" message.
//
// Ordering matters: the engine-nil guard is the FIRST check in the service
// method — it runs before resolveFgaSubject — so even an unauthenticated caller
// gets ErrFgaNotEnabled, NOT Unauthenticated. This test pins that order.
func TestGRPCCheckPermissionsFailClosed(t *testing.T) {
	conn := bootGRPCBufconn(t)
	c := authorizerv1.NewAuthorizerServiceClient(conn)

	// Unauthenticated call (no metadata / bearer): the engine check still fires
	// first, so we expect the FGA-not-enabled error, not Unauthenticated.
	_, err := c.CheckPermissions(context.Background(), &authorizerv1.CheckPermissionsRequest{
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
// up front with service.ErrFgaNotEnabled -> codes.FailedPrecondition.
func TestGRPCListPermissionsFailClosed(t *testing.T) {
	conn := bootGRPCBufconn(t)
	c := authorizerv1.NewAuthorizerServiceClient(conn)

	_, err := c.ListPermissions(context.Background(), &authorizerv1.ListPermissionsRequest{})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status error")
	assert.Equal(t, codes.FailedPrecondition, st.Code(),
		"ErrFgaNotEnabled is a typed FailedPrecondition service error")
	assert.Equal(t, service.ErrFgaNotEnabled.Error(), st.Message())
}

// TestGRPCCheckPermissionsEmptyChecksRejected verifies the protovalidate
// interceptor enforces CheckPermissionsRequest.checks min_items=1. The validate
// interceptor sits OUTSIDE the handler, so an empty checks list is rejected with
// codes.InvalidArgument before the handler (and thus before the engine-nil
// guard) ever runs — the request never reaches the service layer.
func TestGRPCCheckPermissionsEmptyChecksRejected(t *testing.T) {
	conn := bootGRPCBufconn(t)
	c := authorizerv1.NewAuthorizerServiceClient(conn)

	_, err := c.CheckPermissions(context.Background(), &authorizerv1.CheckPermissionsRequest{
		Checks: nil, // empty -> violates min_items: 1
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code(),
		"empty checks must be rejected by the protovalidate interceptor, not the handler")
}
