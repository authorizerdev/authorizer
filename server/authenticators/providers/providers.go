package providers

import (
	"context"
)

// AuthenticatorConfig defines authenticator config
type AuthenticatorConfig struct {
	// ScannerImage is the base64 of QR code image
	ScannerImage string
	// Secrets is the secret key
	Secret string
	// RecoveryCode is the list of recovery codes
	RecoveryCodes []string
	// RecoveryCodeMap is the map of recovery codes
	RecoveryCodeMap map[string]bool
}

// Provider defines authenticators provider
type Provider interface {
	// Generate totp: to generate totp, store secret into db and returns base64 of QR code image
	Generate(ctx context.Context, id string) (*AuthenticatorConfig, error)
	// Validate totp: user passcode with secret stored in our db
	Validate(ctx context.Context, passcode string, userID string) (bool, error)
	// ValidateRecoveryCode totp: allows user to validate using recovery code incase if they lost their device
	ValidateRecoveryCode(ctx context.Context, recoveryCode, userID string) (bool, error)
	// UpdateTotpInfo: to update secret and recovery codes into db and returns base64 of QR code image
	UpdateTotpInfo(ctx context.Context, id string) (*AuthenticatorConfig, error)
}
