package totp

import (
	"github.com/authorizerdev/authorizer/internal/models"
)

type Dependencies struct {
	model models.Provider
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
