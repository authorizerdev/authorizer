package totp

import (
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
)

type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
	// MemoryStoreProvider holds transient, not-yet-confirmed TOTP secrets
	// during a re-enrollment (see Generate/Validate). Deliberately out of the
	// DB so an abandoned re-setup can never desync a working authenticator.
	MemoryStoreProvider memory_store.Provider
	// EncryptionKey is the server-side key used to encrypt TOTP shared
	// secrets at rest. Wired to Config.JWTSecret in internal/authenticators.
	EncryptionKey string
}

type provider struct {
	deps *Dependencies
}

// TOTPConfig defines totp config
type TOTPConfig struct {
	ScannerImage string
	Secret       string
}

// NewProvider returns a new totp provider
func NewProvider(deps *Dependencies) (*provider, error) {
	return &provider{
		deps: deps,
	}, nil
}
