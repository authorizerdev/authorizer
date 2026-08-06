package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authctx"
)

// TestCallerTokenDataCarriesActorID pins that the caller identity is described
// IDENTICALLY on both transports.
//
// callerTokenData has two branches: authctx.Principal (populated only by the
// gRPC interceptor) and the request token (GraphQL/REST). The token branch has
// always carried ActorID; the principal branch dropped it, so an agent's action
// over gRPC was recorded as the user performing it while the same action over
// GraphQL was attributed correctly. A divergence between transports on WHO did
// something is not cosmetic, and nothing failed when it regressed — hence this
// test.
func TestCallerTokenDataCarriesActorID(t *testing.T) {
	p := &provider{}

	t.Run("principal branch preserves the actor", func(t *testing.T) {
		ctx := authctx.WithPrincipal(context.Background(), &authctx.Principal{
			UserID:      "alice",
			LoginMethod: "basic_auth",
			Nonce:       "nonce-1",
			ActorID:     "agent-client-id",
		})
		data, err := p.callerTokenData(ctx, RequestMetadata{})
		require.NoError(t, err)
		require.NotNil(t, data)

		assert.Equal(t, "alice", data.UserID)
		assert.Equal(t, "agent-client-id", data.ActorID,
			"the gRPC branch must carry the actor, or agent actions on gRPC are audited as the user")
	})

	t.Run("an ordinary principal has no actor", func(t *testing.T) {
		ctx := authctx.WithPrincipal(context.Background(), &authctx.Principal{
			UserID: "alice", LoginMethod: "basic_auth", Nonce: "nonce-2",
		})
		data, err := p.callerTokenData(ctx, RequestMetadata{})
		require.NoError(t, err)
		assert.Empty(t, data.ActorID)
	})
}
