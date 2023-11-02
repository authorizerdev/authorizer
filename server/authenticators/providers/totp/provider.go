package totp

import (
	"context"
)

type provider struct {
	ctx context.Context
}

// NewProvider returns a new totp provider
func NewProvider() (*provider, error) {
	ctx := context.Background()
	return &provider{
		ctx: ctx,
	}, nil
}
