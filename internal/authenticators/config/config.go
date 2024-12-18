package config

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
