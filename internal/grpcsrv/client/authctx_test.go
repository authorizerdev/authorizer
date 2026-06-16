package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestWithBearerToken(t *testing.T) {
	t.Run("empty token is a no-op", func(t *testing.T) {
		ctx := WithBearerToken(context.Background(), "")
		_, ok := metadata.FromOutgoingContext(ctx)
		assert.False(t, ok)
	})

	t.Run("sets authorization metadata", func(t *testing.T) {
		ctx := WithBearerToken(context.Background(), "tok-123")
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer tok-123"}, md.Get("authorization"))
	})
}

func TestWithAdminSecret(t *testing.T) {
	t.Run("empty secret is a no-op", func(t *testing.T) {
		ctx := WithAdminSecret(context.Background(), "")
		_, ok := metadata.FromOutgoingContext(ctx)
		assert.False(t, ok)
	})

	t.Run("sets x-authorizer-admin-secret metadata", func(t *testing.T) {
		ctx := WithAdminSecret(context.Background(), "admin-secret")
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"admin-secret"}, md.Get("x-authorizer-admin-secret"))
	})
}

func TestWithAuthorizerURL(t *testing.T) {
	t.Run("empty url is a no-op", func(t *testing.T) {
		ctx := WithAuthorizerURL(context.Background(), "")
		_, ok := metadata.FromOutgoingContext(ctx)
		assert.False(t, ok)
	})

	t.Run("sets x-authorizer-url metadata", func(t *testing.T) {
		ctx := WithAuthorizerURL(context.Background(), "https://auth.example.com")
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"https://auth.example.com"}, md.Get("x-authorizer-url"))
	})
}

func TestWithCookies(t *testing.T) {
	t.Run("no cookies is a no-op", func(t *testing.T) {
		ctx := WithCookies(context.Background())
		_, ok := metadata.FromOutgoingContext(ctx)
		assert.False(t, ok)
	})

	t.Run("empty cookie values are skipped", func(t *testing.T) {
		ctx := WithCookies(context.Background(), "", "session=abc")
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"session=abc"}, md.Get("cookie"))
	})

	t.Run("sets multiple cookie metadata entries", func(t *testing.T) {
		ctx := WithCookies(context.Background(), "session=abc", "admin=xyz")
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"session=abc", "admin=xyz"}, md.Get("cookie"))
	})
}

func TestAuthHelpersCombine(t *testing.T) {
	ctx := context.Background()
	ctx = WithBearerToken(ctx, "tok-123")
	ctx = WithAuthorizerURL(ctx, "https://auth.example.com")
	ctx = WithAdminSecret(ctx, "admin-secret")
	ctx = WithCookies(ctx, "session=abc")

	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer tok-123"}, md.Get("authorization"))
	assert.Equal(t, []string{"https://auth.example.com"}, md.Get("x-authorizer-url"))
	assert.Equal(t, []string{"admin-secret"}, md.Get("x-authorizer-admin-secret"))
	assert.Equal(t, []string{"session=abc"}, md.Get("cookie"))
}
