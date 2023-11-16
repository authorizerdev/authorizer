package totp

import (
	"context"
)

type provider struct {
	ctx context.Context
}

// TOTPConfig defines totp config
type TOTPConfig struct {
	ScannerImage string
	Secret       string
}

// NewProvider returns a new totp provider
func NewProvider() (*provider, error) {
	ctx := context.Background()
	return &provider{
		ctx: ctx,
	}, nil
}
