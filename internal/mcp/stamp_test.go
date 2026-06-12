package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// TestStampAuth covers the per-dispatch metadata bridge: the configured
// bearer must surface as `authorization` and the configured authorizer URL
// as `x-authorizer-url` (JWT issuer validation resolves the host from it —
// without it the in-process bufconn authority would reject every token).
func TestStampAuth(t *testing.T) {
	t.Run("no bearer, no url is a no-op", func(t *testing.T) {
		s := &Server{}
		ctx := s.stampAuth(context.Background())
		_, ok := metadata.FromOutgoingContext(ctx)
		assert.False(t, ok)
	})

	t.Run("bearer and url are both stamped", func(t *testing.T) {
		s := &Server{bearer: "tok-123", authorizerURL: "https://auth.example.com"}
		ctx := s.stampAuth(context.Background())
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer tok-123"}, md.Get("authorization"))
		assert.Equal(t, []string{"https://auth.example.com"}, md.Get("x-authorizer-url"))
	})

	t.Run("bearer without url stamps only authorization", func(t *testing.T) {
		s := &Server{bearer: "tok-123"}
		ctx := s.stampAuth(context.Background())
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer tok-123"}, md.Get("authorization"))
		assert.Empty(t, md.Get("x-authorizer-url"))
	})
}
