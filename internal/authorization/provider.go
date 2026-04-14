package authorization

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/storage"
)

// Principal represents the entity requesting access.
// It is deliberately agnostic -- a Principal can be a human user,
// a service account (M2M), or an AI agent. The evaluation engine
// does not care about the origin; it only evaluates policies against
// the principal's identity and roles.
type Principal struct {
	// ID is the unique identifier of the principal (user ID, client ID, agent ID).
	ID string
	// Type is the kind of principal: "user", "client", or "agent".
	Type string
	// Roles are the roles assigned to this principal.
	Roles []string
	// MaxScopes is an optional delegation ceiling. If set, the principal
	// can never be granted permissions beyond this set, regardless of
	// what policies say. Format: []string{"resource:scope", ...}.
	// Nil means no ceiling (full access based on policies).
	MaxScopes []string
}

// ResourceScope pairs a resource name with a scope name.
// Used for returning all permissions a principal has.
type ResourceScope struct {
	Resource string `json:"resource"`
	Scope    string `json:"scope"`
}

// CheckResult contains the result of a permission check with debugging info.
type CheckResult struct {
	// Allowed is true if the principal has the requested permission.
	Allowed bool
	// MatchedPolicy is the name of the policy that granted access (empty if denied).
	MatchedPolicy string
}

// Provider defines the authorization evaluation engine interface.
type Provider interface {
	// CheckPermission evaluates whether a principal can perform a scope on a resource.
	CheckPermission(ctx context.Context, principal *Principal, resource string, scope string) (*CheckResult, error)

	// GetPrincipalPermissions returns all granted resource:scope pairs for a principal.
	// Used for JWT embedding and dashboard display.
	GetPrincipalPermissions(ctx context.Context, principal *Principal) ([]ResourceScope, error)

	// InvalidateCache removes cached authorization data.
	// Called by admin mutations when permissions/policies change.
	InvalidateCache(ctx context.Context, prefix string) error
}

// Dependencies carries shared resources for constructing an authorization Provider.
type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
}

// Config holds authorization-specific configuration.
// This is separate from the main config to avoid circular imports.
// The values are passed in from cmd/root.go.
type Config struct {
	// Enforcement is the authorization enforcement mode:
	// "disabled", "permissive", or "enforcing".
	Enforcement string
	// CacheTTL is the cache time-to-live in seconds. 0 disables caching.
	CacheTTL int64
}

// provider implements the Provider interface.
type provider struct {
	config          *Config
	log             *zerolog.Logger
	storageProvider storage.Provider
	cache           *cache
}

// New creates a new authorization provider.
func New(cfg *Config, deps *Dependencies) (Provider, error) {
	p := &provider{
		config:          cfg,
		log:             deps.Log,
		storageProvider: deps.StorageProvider,
		cache:           newCache(cfg.CacheTTL),
	}
	return p, nil
}
