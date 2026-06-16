// Package client provides helpers for attaching Authorizer authentication
// metadata to outgoing gRPC client contexts.
package client

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// WithBearerToken attaches an OAuth2 bearer access token to ctx for gRPC calls.
// transport.MetaFromGRPC forwards the value to session/access-token auth checks.
func WithBearerToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

// WithAdminSecret attaches the super-admin secret for AuthorizerAdminService calls.
// transport.MetaFromGRPC forwards the value to the super-admin auth check.
func WithAdminSecret(ctx context.Context, secret string) context.Context {
	if secret == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-authorizer-admin-secret", secret)
}

// WithAuthorizerURL attaches the Authorizer host URL that minted the bearer token.
// Pure-gRPC callers must set this so issuer validation resolves the correct host.
func WithAuthorizerURL(ctx context.Context, url string) context.Context {
	if url == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-authorizer-url", url)
}

// WithCookies attaches one or more Cookie header lines for session-based gRPC auth.
// Each non-empty cookie is sent as a separate "cookie" metadata entry, matching
// transport.MetaFromGRPC cookiesFromMetadata parsing.
func WithCookies(ctx context.Context, cookies ...string) context.Context {
	for _, cookie := range cookies {
		if cookie == "" {
			continue
		}
		ctx = metadata.AppendToOutgoingContext(ctx, "cookie", cookie)
	}
	return ctx
}
