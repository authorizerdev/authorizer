package totp

import (
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/data_store"
)

type Dependencies struct {
	Log *zerolog.Logger
	DB  data_store.Provider
}

type provider struct {
	deps Dependencies
}

// TOTPConfig defines totp config
type TOTPConfig struct {
	ScannerImage string
	Secret       string
}

// NewProvider returns a new totp provider
func NewProvider(deps Dependencies) (*provider, error) {
	return &provider{
		deps: deps,
	}, nil
}
