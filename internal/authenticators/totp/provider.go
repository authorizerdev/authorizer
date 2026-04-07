package totp

import (
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/storage"
)

type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
	// EncryptionKey is the server-side key used to encrypt TOTP shared
	// secrets at rest. Wired to Config.JWTSecret in internal/authenticators.
	EncryptionKey string
	// EnableLazyMigration opts in to rewriting legacy plaintext TOTP rows
	// into the enc:v1: ciphertext format on the first successful Validate.
	//
	// MUST stay false during a rolling deploy from a pre-encryption
	// release. Old replicas do not understand the enc:v1: prefix and will
	// treat a migrated row as a base32 secret, locking out users routed
	// to those replicas. Recommended sequence:
	//
	//   1. Roll out this release to ALL replicas with the flag OFF
	//      (default). Validate continues to work for both legacy and
	//      already-encrypted rows because the read path transparently
	//      passes legacy plaintext through.
	//   2. Confirm every replica is on the new version.
	//   3. Restart with --enable-totp-migration=true. Subsequent
	//      validations rewrite legacy rows into enc:v1: form.
	//   4. After enough time for active users to have validated at least
	//      once, the flag can be left on indefinitely — it is a no-op
	//      for already-encrypted rows.
	//
	// DEPRECATED: the whole lazy-migration code path is intended for
	// removal in vN+2.
	EnableLazyMigration bool
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
