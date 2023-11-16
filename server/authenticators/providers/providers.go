package providers

import "context"

// AuthenticatorConfig defines authenticator config
type AuthenticatorConfig struct {
	// ScannerImage is the base64 of QR code image
	ScannerImage string
	// Secrets is the secret key
	Secret string
	// RecoveryCode is the secret key
	RecoveryCodes []string
}

// Provider defines authenticators provider
type Provider interface {
	// Generate totp: to generate totp, store secret into db and returns base64 of QR code image
	Generate(ctx context.Context, id string) (*AuthenticatorConfig, error)
	// Validate totp: user passcode with secret stored in our db
	Validate(ctx context.Context, passcode string, id string) (bool, error)
	// RecoveryCode totp: gives a recovery code for first time user
	RecoveryCode(ctx context.Context, id string) (*string, error)
}
