package interceptors

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// info builds a *grpc.UnaryServerInfo for a fake RPC. The full-method name is
// the only field interceptors actually read.
func info(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: method}
}

func TestRecovery_TurnsPanicIntoInternal(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)

	r := Recovery(&log)
	_, err := r(context.Background(), nil, info("/svc/Method"), func(_ context.Context, _ any) (any, error) {
		panic("kaboom")
	})

	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status error")
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal server error", st.Message(), "panic detail must not leak to clients")
	out := buf.String()
	// The stack + type get logged. The recovered VALUE does NOT — security
	// audit H2: panic values can carry credentials (Password / RefreshToken
	// / OTP / AdminSecret) that must not reach the log stream.
	assert.Contains(t, out, "panicked")
	assert.Contains(t, out, `"panic_type":"string"`)
	assert.NotContains(t, out, "kaboom",
		"the panic VALUE must not appear in logs; only its type — see H2")
}

// TestRecovery_DoesNotLogCredentialBearingPanicValue is the regression test
// for security audit H2: a handler that panics with a value containing
// credentials must NOT have those credentials written to the log stream.
func TestRecovery_DoesNotLogCredentialBearingPanicValue(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	r := Recovery(&log)
	_, _ = r(context.Background(), nil, info("/svc/X"), func(_ context.Context, _ any) (any, error) {
		// Simulate a handler panicking with a credential-bearing value.
		panic("password=hunter2 token=secretXYZ")
	})
	out := buf.String()
	assert.NotContains(t, out, "hunter2", "panic value must not reach logs")
	assert.NotContains(t, out, "secretXYZ", "panic value must not reach logs")
	assert.Contains(t, out, `"panic_type":"string"`, "type should still be logged for triage")
}

func TestRecovery_PassesNormalErrorsThrough(t *testing.T) {
	log := zerolog.Nop()
	r := Recovery(&log)
	want := status.Error(codes.NotFound, "no")
	_, err := r(context.Background(), nil, info("/svc/X"), func(_ context.Context, _ any) (any, error) {
		return nil, want
	})
	assert.Equal(t, want, err)
}

func TestLogging_OkPath(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	mw := Logging(&log)
	_, err := mw(context.Background(), nil, info("/svc/Foo"), func(_ context.Context, _ any) (any, error) {
		return "ok", nil
	})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, `"method":"/svc/Foo"`)
	assert.Contains(t, out, `"code":"OK"`)
	assert.Contains(t, out, `"level":"info"`)
}

func TestLogging_ErrorPathRaisesLevel(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	mw := Logging(&log)
	_, _ = mw(context.Background(), nil, info("/svc/Bad"), func(_ context.Context, _ any) (any, error) {
		return nil, status.Error(codes.Internal, "boom")
	})
	out := buf.String()
	assert.Contains(t, out, `"code":"Internal"`)
	assert.Contains(t, out, `"level":"error"`, "Internal/Unknown/DataLoss must raise log level to error")
}

func TestLogging_PermissionDeniedIsWarn(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf)
	mw := Logging(&log)
	_, _ = mw(context.Background(), nil, info("/svc/X"), func(_ context.Context, _ any) (any, error) {
		return nil, status.Error(codes.PermissionDenied, "no")
	})
	assert.Contains(t, buf.String(), `"level":"warn"`, "non-Internal failures must log at warn, not error")
}

func TestValidate_RejectsBadRequest(t *testing.T) {
	mw, err := Validate()
	require.NoError(t, err)

	// RevokeRequest enforces refresh_token min_len=1 via buf.validate.field
	// — an empty string should fail the interceptor before any handler runs.
	req := &authorizerv1.RevokeRequest{RefreshToken: ""}
	_, err = mw(context.Background(), req, info("/authorizer.v1.Authorizer/Revoke"), func(_ context.Context, _ any) (any, error) {
		t.Fatal("handler must NOT run for an invalid request")
		return nil, nil
	})
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestValidate_AllowsValidRequest(t *testing.T) {
	mw, err := Validate()
	require.NoError(t, err)
	called := false
	_, err = mw(context.Background(), &authorizerv1.MetaRequest{}, info("/authorizer.v1.Authorizer/Meta"), func(_ context.Context, _ any) (any, error) {
		called = true
		return &authorizerv1.MetaResponse{}, nil
	})
	require.NoError(t, err)
	assert.True(t, called, "valid request must reach the handler")
}

func TestValidate_NonProtoRequestPassesThrough(t *testing.T) {
	mw, err := Validate()
	require.NoError(t, err)
	_, err = mw(context.Background(), "not-a-proto", info("/svc/X"), func(_ context.Context, _ any) (any, error) {
		return nil, nil
	})
	require.NoError(t, err, "non-proto requests must not be rejected by the validator")
}

// TestValidate_PreservesInvariant guards against regressions where someone
// makes Validate() return a non-functional middleware (e.g. by reordering
// the protovalidate.New() call). If the validator itself fails to build,
// callers must learn about it at startup, not at first request.
func TestValidate_BuildsCleanly(t *testing.T) {
	mw, err := Validate()
	require.NoError(t, err)
	require.NotNil(t, mw)
	// Sanity check: the returned interceptor type is what gRPC expects.
	_ = grpc.UnaryServerInterceptor(mw)
}
