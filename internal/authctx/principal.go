// Package authctx carries authentication principal details on context.Context.
package authctx

import "strings"

import "context"

type principalContextKey struct{}

// Principal is the authenticated caller identity resolved by transport auth.
type Principal struct {
	UserID       string
	LoginMethod  string
	Nonce        string
	IsSuperAdmin bool
	// ActorID is the immediate actor of an RFC 8693 delegated token — the
	// agent's client_id from `act.sub`. Empty for first-party callers.
	//
	// UserID stays the delegating user: the request IS being made for them.
	// ActorID records WHO is making it, which is the distinction RFC 8693 §1.1
	// draws between delegation ("A representing B", A keeps its own identity)
	// and impersonation ("A is indistinguishable from B"). Without it a
	// delegated action is attributed to the human, which is both an audit lie
	// and the Confused Deputy precondition.
	ActorID string
	// Scope is the token's `scope` claim, carried so the gRPC interceptor can
	// enforce per-operation scope for delegated callers. See
	// internal/delegatedscope.
	Scope []string
}

// IsDelegated reports whether this principal is an agent acting for a user.
func (p *Principal) IsDelegated() bool {
	return p != nil && strings.TrimSpace(p.ActorID) != ""
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
