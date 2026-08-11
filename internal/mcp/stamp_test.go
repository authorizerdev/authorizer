package mcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// TestStampAuth covers the per-dispatch metadata bridge. The stdio cases below
// pass a nil header (there is no HTTP request), so the process-wide bearer and
// authorizer URL are what surface; TestStampAuthPerRequest covers the HTTP case,
// where the caller's own Authorization header must win and nothing else may
// cross.
func TestStampAuth(t *testing.T) {
	t.Run("no bearer, no url is a no-op", func(t *testing.T) {
		s := &Server{}
		ctx := s.stampAuth(context.Background(), nil)
		_, ok := metadata.FromOutgoingContext(ctx)
		assert.False(t, ok)
	})

	t.Run("bearer and url are both stamped", func(t *testing.T) {
		s := &Server{bearer: "tok-123", authorizerURL: "https://auth.example.com"}
		ctx := s.stampAuth(context.Background(), nil)
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer tok-123"}, md.Get("authorization"))
		assert.Equal(t, []string{"https://auth.example.com"}, md.Get("x-authorizer-url"))
	})

	t.Run("bearer without url stamps only authorization", func(t *testing.T) {
		s := &Server{bearer: "tok-123"}
		ctx := s.stampAuth(context.Background(), nil)
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer tok-123"}, md.Get("authorization"))
		assert.Empty(t, md.Get("x-authorizer-url"))
	})
}

// TestStampAuthPerRequest pins the property that makes one HTTP MCP server able
// to serve many callers: identity comes from the request, not the process.
//
// It also pins what must NOT cross. transport.MetaFromGRPC reconstructs cookies
// and x-authorizer-url from gRPC metadata, so forwarding headers wholesale would
// hand the auth path of an audience-bound surface a browser session cookie or a
// caller-chosen host. The interceptor refuses those anyway, but the bridge must
// not offer them.
func TestStampAuthPerRequest(t *testing.T) {
	t.Run("the request's Authorization wins over the static bearer", func(t *testing.T) {
		s := &Server{bearer: "process-wide", authorizerURL: "https://auth.example.com"}
		h := http.Header{}
		h.Set("Authorization", "Bearer caller-token")

		md, ok := metadata.FromOutgoingContext(s.stampAuth(context.Background(), h))
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer caller-token"}, md.Get("authorization"),
			"an HTTP caller must be attributed to themselves, never to whoever started the process")
		assert.Empty(t, md.Get("x-authorizer-url"),
			"--url is mandatory for HTTP MCP, so a host hint is at best redundant and at worst caller-controlled")
	})

	t.Run("cookies and other headers never cross", func(t *testing.T) {
		s := &Server{}
		h := http.Header{}
		h.Set("Authorization", "Bearer caller-token")
		h.Set("Cookie", "authorizer_session=abc")
		h.Set("X-Authorizer-URL", "https://evil.example.com")
		h.Set("X-Authorizer-Admin-Secret", "hunter2")

		md, ok := metadata.FromOutgoingContext(s.stampAuth(context.Background(), h))
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer caller-token"}, md.Get("authorization"))
		assert.Empty(t, md.Get("cookie"))
		assert.Empty(t, md.Get("x-authorizer-url"))
		assert.Empty(t, md.Get("x-authorizer-admin-secret"))
		assert.Len(t, md, 1, "exactly one header crosses the bridge")
	})

	t.Run("an empty Authorization falls back to the stdio bearer", func(t *testing.T) {
		s := &Server{bearer: "process-wide"}
		md, ok := metadata.FromOutgoingContext(s.stampAuth(context.Background(), http.Header{}))
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer process-wide"}, md.Get("authorization"))
	})
}
