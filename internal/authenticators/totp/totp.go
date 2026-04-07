package totp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"github.com/authorizerdev/authorizer/internal/authenticators/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Generate generates a Time-Based One-Time Password (TOTP) for a user and returns the base64-encoded QR code for frontend display.
func (p *provider) Generate(ctx context.Context, id string) (*config.AuthenticatorConfig, error) {
	log := p.deps.Log.With().Str("func", "Generate (totp provider)").Logger()
	var buf bytes.Buffer
	// Get user details
	user, err := p.deps.StorageProvider.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Generate totp, Authenticators hash is valid for 30 seconds
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "authorizer",
		AccountName: refs.StringValue(user.Email),
	})
	if err != nil {
		return nil, err
	}
	// Generating image for key and encoding to base64 for displaying in frontend
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, err
	}
	png.Encode(&buf, img)
	encodedText := crypto.EncryptB64(buf.String())
	secret := key.Secret()
	recoveryCodes := []string{}
	for i := 0; i < 10; i++ {
		recoveryCodes = append(recoveryCodes, uuid.NewString())
	}
	// Converting recoveryCodes to string
	recoverCodesMap := map[string]bool{}
	for i := 0; i < len(recoveryCodes); i++ {
		recoverCodesMap[recoveryCodes[i]] = false
	}
	// Converting recoveryCodesMap to string
	jsonData, err := json.Marshal(recoverCodesMap)
	if err != nil {
		return nil, err
	}
	recoveryCodesString := string(jsonData)
	// Encrypt the TOTP shared secret at rest. The plaintext `secret` is
	// returned to the caller (frontend needs it to display the QR code
	// for enrollment) but never written to storage in plaintext.
	encryptedSecret, err := crypto.EncryptTOTPSecret(secret, p.deps.EncryptionKey)
	if err != nil {
		return nil, err
	}
	totpModel := &schemas.Authenticator{
		Secret:        encryptedSecret,
		RecoveryCodes: refs.NewStringRef(recoveryCodesString),
		UserID:        user.ID,
		Method:        constants.EnvKeyTOTPAuthenticator,
	}
	authenticator, err := p.deps.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		log.Debug().Err(err).Msg("error getting authenticator details")
		// continue
	}
	if authenticator == nil {
		// if authenticator is nil then create new authenticator
		_, err = p.deps.StorageProvider.AddAuthenticator(ctx, totpModel)
		if err != nil {
			return nil, err
		}
	} else {
		authenticator.Secret = encryptedSecret
		authenticator.RecoveryCodes = refs.NewStringRef(recoveryCodesString)
		// if authenticator is not nil then update authenticator
		_, err = p.deps.StorageProvider.UpdateAuthenticator(ctx, authenticator)
		if err != nil {
			return nil, err
		}
	}
	return &config.AuthenticatorConfig{
		ScannerImage:    encodedText,
		Secret:          secret,
		RecoveryCodes:   recoveryCodes,
		RecoveryCodeMap: recoverCodesMap,
	}, nil
}

// Validate validates a Time-Based One-Time Password (TOTP) against the stored TOTP secret for a user.
func (p *provider) Validate(ctx context.Context, passcode string, userID string) (bool, error) {
	// get totp details
	totpModel, err := p.deps.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return false, err
	}
	// Decrypt the stored secret. DecryptTOTPSecret transparently handles
	// both new ciphertext rows (enc:v1: prefix) and legacy plaintext rows
	// from a pre-encryption release, so an upgrade does not break in-flight
	// users.
	plainSecret, err := crypto.DecryptTOTPSecret(totpModel.Secret, p.deps.EncryptionKey)
	if err != nil {
		return false, err
	}
	// validate totp
	status := totp.Validate(passcode, plainSecret)

	// Lazy migration: if the stored value is still legacy plaintext and
	// the user just produced a valid TOTP, re-encrypt and persist before
	// the next verification. Operators upgrading from a pre-encryption
	// release have all enrolled secrets converted to enc:v1: form on
	// first successful login. The auto-migration code is DEPRECATED and
	// will be removed two minor versions after this lands.
	needsLazyMigration := status && !crypto.IsEncryptedTOTPSecret(totpModel.Secret)

	// checks if user not signed in for totp and totp code is correct then VerifiedAt will be stored in db
	if totpModel.VerifiedAt == nil && status {
		timeNow := time.Now().Unix()
		totpModel.VerifiedAt = &timeNow
	}
	if needsLazyMigration {
		ct, encErr := crypto.EncryptTOTPSecret(plainSecret, p.deps.EncryptionKey)
		if encErr == nil {
			totpModel.Secret = ct
		}
	}
	if needsLazyMigration || (totpModel.VerifiedAt != nil && status) {
		if _, err = p.deps.StorageProvider.UpdateAuthenticator(ctx, totpModel); err != nil {
			return false, err
		}
	}
	return status, nil
}

// ValidateRecoveryCode validates a Time-Based One-Time Password (TOTP) recovery code against the stored TOTP recovery code for a user.
func (p *provider) ValidateRecoveryCode(ctx context.Context, recoveryCode, userID string) (bool, error) {
	// get totp details
	totpModel, err := p.deps.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return false, err
	}
	// convert recoveryCodes to map
	recoveryCodesMap := map[string]bool{}
	err = json.Unmarshal([]byte(refs.StringValue(totpModel.RecoveryCodes)), &recoveryCodesMap)
	if err != nil {
		return false, err
	}
	// check if recovery code is valid
	if val, ok := recoveryCodesMap[recoveryCode]; !ok {
		return false, fmt.Errorf("invalid recovery code")
	} else if val {
		return false, fmt.Errorf("recovery code already used")
	}
	// update recovery code map
	recoveryCodesMap[recoveryCode] = true
	// convert recoveryCodesMap to string
	jsonData, err := json.Marshal(recoveryCodesMap)
	if err != nil {
		return false, err
	}
	recoveryCodesString := string(jsonData)
	totpModel.RecoveryCodes = refs.NewStringRef(recoveryCodesString)
	// update recovery code map in db
	_, err = p.deps.StorageProvider.UpdateAuthenticator(ctx, totpModel)
	if err != nil {
		return false, err
	}
	return true, nil
}
