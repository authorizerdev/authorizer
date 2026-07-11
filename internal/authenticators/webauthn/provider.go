package webauthn

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies are injected into the webauthn provider, mirroring the totp
// provider's construction pattern.
type Dependencies struct {
	Log                 *zerolog.Logger
	StorageProvider     storage.Provider
	MemoryStoreProvider memory_store.Provider
}

// Provider wraps github.com/go-webauthn/webauthn as an Authorizer Relying
// Party. It owns the two-step ceremony state (the challenge held in the memory
// store between the *_options and *_verify calls) and the persistence of
// registered credentials. Higher-level policy (email-verification gate, token
// issuance) lives in internal/service.
type Provider interface {
	// BeginRegistration starts a registration ceremony for user, stores the
	// challenge, and returns PublicKeyCredentialCreationOptions as JSON.
	BeginRegistration(ctx context.Context, host string, user *schemas.User) (string, error)
	// FinishRegistration verifies the attestation response and persists a new
	// credential owned by user with the given (optional) name.
	FinishRegistration(ctx context.Context, host string, user *schemas.User, name, responseJSON string) (*schemas.WebauthnCredential, error)
	// BeginLogin starts a login ceremony scoped to user's own credentials
	// (MFA-alternative flow) and returns PublicKeyCredentialRequestOptions JSON.
	BeginLogin(ctx context.Context, host string, user *schemas.User) (string, error)
	// BeginDiscoverableLogin starts a usernameless login ceremony with empty
	// allowCredentials so the browser surfaces any resident passkey.
	BeginDiscoverableLogin(ctx context.Context, host string) (string, error)
	// FinishLogin verifies an assertion and returns the authenticated user and
	// credential. It handles both ceremonies transparently: usernameless (the
	// stored session has no user, the user is resolved from the credential id)
	// and scoped (the session pins the user). On success the credential's
	// sign-count/flags/last-used are persisted.
	FinishLogin(ctx context.Context, host, responseJSON string) (*schemas.User, *schemas.WebauthnCredential, error)
}

type provider struct {
	deps *Dependencies
}

// NewProvider returns a new webauthn provider.
func NewProvider(deps *Dependencies) (*provider, error) {
	return &provider{deps: deps}, nil
}
