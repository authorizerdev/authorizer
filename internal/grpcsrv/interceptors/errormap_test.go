package interceptors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage"
)

var nopLog = zerolog.Nop()

// TestErrorMap_MasksStorageError is the regression guard for issue #356: raw
// driver text from the storage layer must never reach a gRPC (or, through the
// gateway, REST) client. The client gets codes.Internal with a generic message
// and error_id; the original error is logged under the same error_id.
func TestErrorMap_MasksStorageError(t *testing.T) {
	const driverMsg = "mssql: Cannot insert duplicate key row in object 'dbo.authorizer_users' with unique index 'idx_email'"
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	mw := ErrorMap(&log)
	_, err := mw(context.Background(), nil, info("/authorizer.v1.AuthorizerService/Signup"), func(_ context.Context, _ any) (any, error) {
		return nil, fmt.Errorf("signup: %w", &storage.Error{Err: errors.New(driverMsg)})
	})

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.NotContains(t, st.Message(), "mssql")
	assert.Regexp(t, `^internal server error \(error_id: \S+\)$`, st.Message())

	logged := buf.String()
	assert.Contains(t, logged, driverMsg, "original error must be logged server-side")
	assert.Contains(t, logged, "/authorizer.v1.AuthorizerService/Signup")
	id := st.Message()[len("internal server error (error_id: ") : len(st.Message())-1]
	assert.Contains(t, logged, `"error_id":"`+id+`"`)
}

// TestErrorMap_TypedErrorWrappingStorageErrorKeepsMessage: a service that
// deliberately classified a storage failure keeps its chosen message.
func TestErrorMap_TypedErrorWrappingStorageErrorKeepsMessage(t *testing.T) {
	err := runErrorMap(t, fmt.Errorf("%w: %w", service.NotFound("user not found"), &storage.Error{Err: errors.New("record not found")}))
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "user not found", st.Message())
}

// runErrorMap feeds handlerErr through the ErrorMap interceptor and returns the
// mapped error, exactly as a gRPC/REST client would receive it.
func runErrorMap(t *testing.T, handlerErr error) error {
	t.Helper()
	mw := ErrorMap(&nopLog)
	_, err := mw(context.Background(), nil, info("/authorizer.v1.AuthorizerAdminService/CreateOrganization"), func(_ context.Context, _ any) (any, error) {
		return nil, handlerErr
	})
	return err
}

// TestErrorMap_KindToCode locks the transport-neutral Kind -> gRPC code mapping,
// including the newly-added AlreadyExists -> codes.AlreadyExists (HTTP 409).
func TestErrorMap_KindToCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"invalid argument", service.InvalidArgument("name is required"), codes.InvalidArgument},
		{"already exists", service.AlreadyExists("an organization with this name already exists"), codes.AlreadyExists},
		{"not found", service.NotFound("organization not found"), codes.NotFound},
		{"failed precondition", service.FailedPrecondition("email sending is disabled"), codes.FailedPrecondition},
		{"too many requests", service.TooManyRequests("slow down"), codes.ResourceExhausted},
		{"unauthenticated", service.Unauthenticated("unauthorized"), codes.Unauthenticated},
		{"permission denied", service.PermissionDenied("nope"), codes.PermissionDenied},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := runErrorMap(t, c.err)
			st, ok := status.FromError(err)
			require.True(t, ok, "expected a gRPC status error")
			assert.Equal(t, c.want, st.Code())
			assert.Equal(t, c.err.Error(), st.Message(), "message text must be preserved verbatim")
		})
	}
}

// TestErrorMap_AlreadyExistsIsNot500 is the regression guard for the previously
// unimplemented ALREADY_EXISTS contract: a conflict must surface as 409-class
// AlreadyExists, never as Internal/500.
func TestErrorMap_AlreadyExistsIsNot500(t *testing.T) {
	err := runErrorMap(t, service.AlreadyExists("issuer_url already registered: https://idp.example.com"))
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.NotEqual(t, codes.Internal, st.Code())
}

// TestErrorMap_WrappedTypedErrorKeepsKind mirrors the admin SAML/OIDC helpers:
// a typed validation error wrapped with extra field context via %w must still
// map to InvalidArgument (errors.As reaches the inner Kind), not Internal.
func TestErrorMap_WrappedTypedErrorKeepsKind(t *testing.T) {
	wrapped := fmt.Errorf("idp_sso_url %w", service.InvalidArgument("must be a valid https URL"))
	err := runErrorMap(t, wrapped)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// TestErrorMap_BareErrorIsInternal confirms unclassified errors (storage
// failures, secret generation, etc.) remain Internal — the correct default.
func TestErrorMap_BareErrorIsInternal(t *testing.T) {
	err := runErrorMap(t, errors.New("failed to store connection"))
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

// TestErrorMap_NilPassesThrough confirms the happy path is untouched.
func TestErrorMap_NilPassesThrough(t *testing.T) {
	mw := ErrorMap(&nopLog)
	resp, err := mw(context.Background(), nil, info("/svc/M"), func(_ context.Context, _ any) (any, error) {
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}
