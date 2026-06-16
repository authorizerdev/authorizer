// Package authctx carries authentication principal details on context.Context.
package authctx

import "context"

type principalContextKey struct{}

// Principal is the authenticated caller identity resolved by transport auth.
type Principal struct {
	UserID       string
	LoginMethod  string
	Nonce        string
	IsSuperAdmin bool
}

// WithPrincipal stores p in ctx and returns the derived context.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, p)
}

// FromContext returns the principal stored on ctx.
func FromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(*Principal)
	if !ok || p == nil {
		return nil, false
	}
	return p, true
}
