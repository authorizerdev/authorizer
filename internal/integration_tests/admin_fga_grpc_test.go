package integration_tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// The admin FGA RPC tests run against the standard admin test harness, which
// wires NO authorization engine (getTestConfig sets no --fga-store). That gives
// every op two deterministic assertions without standing up a real FGA store:
//
//   - AUTH fail-closed: no admin secret => codes.Unauthenticated.
//   - ENGINE fail-closed: with a valid admin secret but no engine configured =>
//     codes.FailedPrecondition (service.ErrFgaNotEnabled). This is the
//     security-critical "fail closed when FGA is not configured" contract — the
//     ops must never nil-deref or silently succeed.

// TestAdminFgaGetModelGRPC exercises AuthorizerAdminService.FgaGetModel over
// gRPC: the auth fail-closed contract and the engine-not-configured contract.
func TestAdminFgaGetModelGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.FgaGetModel(context.Background(), &authorizerv1.FgaGetModelRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
		_, err := client.FgaGetModel(adminCtx(cfg.AdminSecret), &authorizerv1.FgaGetModelRequest{})
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

// TestAdminFgaWriteModelGRPC exercises AuthorizerAdminService.FgaWriteModel over
// gRPC: the auth fail-closed contract and the engine-not-configured contract.
func TestAdminFgaWriteModelGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.FgaWriteModel(context.Background(), &authorizerv1.FgaWriteModelRequest{
			Dsl: "model\n  schema 1.1\ntype user",
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
		_, err := client.FgaWriteModel(adminCtx(cfg.AdminSecret), &authorizerv1.FgaWriteModelRequest{
			Dsl: "model\n  schema 1.1\ntype user",
		})
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

// TestAdminFgaWriteTuplesGRPC exercises AuthorizerAdminService.FgaWriteTuples
// over gRPC: the auth fail-closed contract and the engine-not-configured
// contract.
func TestAdminFgaWriteTuplesGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	tuples := []*authorizerv1.FgaTupleInput{
		{User: "user:alice", Relation: "owner", Object: "document:1"},
	}

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.FgaWriteTuples(context.Background(), &authorizerv1.FgaWriteTuplesRequest{Tuples: tuples})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
		_, err := client.FgaWriteTuples(adminCtx(cfg.AdminSecret), &authorizerv1.FgaWriteTuplesRequest{Tuples: tuples})
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

// TestAdminFgaDeleteTuplesGRPC exercises AuthorizerAdminService.FgaDeleteTuples
// over gRPC: the auth fail-closed contract and the engine-not-configured
// contract.
func TestAdminFgaDeleteTuplesGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	tuples := []*authorizerv1.FgaTupleInput{
		{User: "user:alice", Relation: "owner", Object: "document:1"},
	}

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.FgaDeleteTuples(context.Background(), &authorizerv1.FgaDeleteTuplesRequest{Tuples: tuples})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
		_, err := client.FgaDeleteTuples(adminCtx(cfg.AdminSecret), &authorizerv1.FgaDeleteTuplesRequest{Tuples: tuples})
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

// TestAdminFgaReadTuplesGRPC exercises AuthorizerAdminService.FgaReadTuples over
// gRPC: the auth fail-closed contract and the engine-not-configured contract.
func TestAdminFgaReadTuplesGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.FgaReadTuples(context.Background(), &authorizerv1.FgaReadTuplesRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
		_, err := client.FgaReadTuples(adminCtx(cfg.AdminSecret), &authorizerv1.FgaReadTuplesRequest{})
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

// TestAdminFgaListUsersGRPC exercises AuthorizerAdminService.FgaListUsers over
// gRPC: the auth fail-closed contract and the engine-not-configured contract.
func TestAdminFgaListUsersGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	req := &authorizerv1.FgaListUsersRequest{
		Object:   "document:1",
		Relation: "viewer",
		UserType: "user",
	}

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.FgaListUsers(context.Background(), req)
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
		_, err := client.FgaListUsers(adminCtx(cfg.AdminSecret), req)
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

// TestAdminFgaExpandGRPC exercises AuthorizerAdminService.FgaExpand over gRPC:
// the auth fail-closed contract and the engine-not-configured contract.
func TestAdminFgaExpandGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	req := &authorizerv1.FgaExpandRequest{
		Relation: "viewer",
		Object:   "document:1",
	}

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.FgaExpand(context.Background(), req)
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
		_, err := client.FgaExpand(adminCtx(cfg.AdminSecret), req)
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}

// TestAdminFgaResetGRPC exercises AuthorizerAdminService.FgaReset over gRPC: the
// auth fail-closed contract and the engine-not-configured contract.
func TestAdminFgaResetGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.FgaReset(context.Background(), &authorizerv1.FgaResetRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
		_, err := client.FgaReset(adminCtx(cfg.AdminSecret), &authorizerv1.FgaResetRequest{})
		require.Error(t, err)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	})
}
