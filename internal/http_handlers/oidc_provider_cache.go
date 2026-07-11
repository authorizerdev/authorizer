package http_handlers

import (
	"context"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

// oidcProviderCache memoizes *oidc.Provider by issuer URL. Each miss performs a
// network round-trip to fetch the issuer's /.well-known/openid-configuration;
// discovery documents are effectively static, so the provider is created once
// per issuer and reused across logins instead of on every OAuth callback.
//
// ponytail: global cache, no TTL — add refresh if an IdP ever rotates its discovery doc.
var oidcProviderCache sync.Map // issuer(string) -> *oidc.Provider

// getOIDCProvider returns a cached *oidc.Provider for the issuer, creating and
// caching it on first use.
func getOIDCProvider(ctx context.Context, issuer string) (*oidc.Provider, error) {
	if p, ok := oidcProviderCache.Load(issuer); ok {
		return p.(*oidc.Provider), nil
	}
	p, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	// LoadOrStore so concurrent first-time misses converge on one instance.
	actual, _ := oidcProviderCache.LoadOrStore(issuer, p)
	return actual.(*oidc.Provider), nil
}
