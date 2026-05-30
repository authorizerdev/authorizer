package service

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

// Dependencies are the subsystems a Provider needs. The set will grow as
// more operations migrate from internal/graphql into this package.
type Dependencies struct {
	Log *zerolog.Logger

	AuditProvider       audit.Provider
	EmailProvider       email.Provider
	EventsProvider      events.Provider
	MemoryStoreProvider memory_store.Provider
	SMSProvider         sms.Provider
	StorageProvider     storage.Provider
	TokenProvider       token.Provider
}

// Provider is the transport-agnostic API for Authorizer public operations.
// Each method takes the inbound RequestMetadata and returns a typed response
// plus a ResponseSideEffects describing cookies (and other transport
// artifacts) the caller must apply.
//
// During the staged migration from internal/graphql, this interface grows one
// method per phase. Operations not yet migrated continue to live as
// graphqlProvider methods until they're moved here.
type Provider interface {
	// SignUp registers a new user. Public — no authentication required.
	// Permissions: none.
	SignUp(ctx context.Context, meta RequestMetadata, params *model.SignUpRequest) (*model.AuthResponse, *ResponseSideEffects, error)

	// Meta returns server discovery information (feature flags + provider
	// availability). Public — no authentication required.
	// Permissions: none.
	Meta(ctx context.Context, meta RequestMetadata) (*model.Meta, *ResponseSideEffects, error)
}

// New constructs a new service provider.
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	return &provider{
		Config:       cfg,
		Dependencies: *deps,
	}, nil
}

type provider struct {
	*config.Config
	Dependencies
}

// Compile-time check that provider satisfies Provider.
var _ Provider = (*provider)(nil)
