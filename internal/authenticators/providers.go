package authenticators

import (
	"context"

	ac "github.com/authorizerdev/authorizer/internal/authenticators/config"
	"github.com/authorizerdev/authorizer/internal/authenticators/totp"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/rs/zerolog"
)

// Dependencies defines the dependencies for authenticators provider
type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
}

// Provider defines authenticators provider
type Provider interface {
	// Generate totp: to generate totp, store secret into db and returns base64 of QR code image
	Generate(ctx context.Context, id string) (*ac.AuthenticatorConfig, error)
	// Validate totp: user passcode with secret stored in our db
	Validate(ctx context.Context, passcode string, userID string) (bool, error)
	// ValidateRecoveryCode totp: allows user to validate using recovery code incase if they lost their device
	ValidateRecoveryCode(ctx context.Context, recoveryCode, userID string) (bool, error)
}

// New returns a new authenticators provider
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	if cfg.DisableTOTPLogin {
		return nil, nil
	}
	return totp.NewProvider(&totp.Dependencies{
		Log:             deps.Log,
		StorageProvider: deps.StorageProvider,
	})
}
