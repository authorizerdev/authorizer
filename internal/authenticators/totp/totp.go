package totp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// Validate validates a Time-Based One-Time Password (TOTP) against the
// stored TOTP secret for a user.
//
// The stored value can be in either of two forms:
//
//  1. enc:v1:<ciphertext> — the at-rest format. Decrypt and use the
//     plaintext to compute the expected code.
//  2. <raw base32> — a legacy row written by a pre-encryption release.
//     Use the stored value directly as the secret. On a successful
//     validation the row is re-encrypted in place (best-effort) so the
//     next read takes the encrypted path.
//
// Concurrency: two replicas observing the same legacy row may both
// decrypt, re-encrypt, and write before either commits. The two writes
// carry the same plaintext under different AES-GCM nonces; whichever
// lands last wins, the contents are semantically identical, and the row
// is permanently in the enc:v1: form afterwards. Subsequent calls take
// the encrypted-path branch immediately.
//
// Rolling-deploy note: a replica still on the pre-encryption release
// will read a migrated row as if it were a base32 secret and fail. For a
// rolling rollout across multiple replicas, prefer to complete the
// rollout before any TOTP user logs in (e.g. with a brief maintenance
// window) — or use an atomic deploy.
//
// Best-effort write: a failure to update the authenticator row after a
// successful validation never fails the login. The user supplied a valid
// TOTP code; refusing it because of a transient DB error or migration
// encrypt failure would be a worse outcome than a delayed VerifiedAt or
// a delayed migration. Failures are logged with structured fields.
func (p *provider) Validate(ctx context.Context, passcode string, userID string) (bool, error) {
	log := p.deps.Log.With().Str("func", "totp.Validate").Str("user_id", userID).Logger()

	totpModel, err := p.deps.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return false, err
	}

	var (
		plainSecret string
		// migrate is set when the stored value is legacy plaintext AND
		// the validation succeeds — at that point we know the raw value
		// is a real base32 secret worth re-encrypting in place.
		migrate bool
	)

	plainSecret, decErr := crypto.DecryptTOTPSecret(totpModel.Secret, p.deps.EncryptionKey)
	switch {
	case decErr == nil:
		// enc:v1: row — use the decrypted plaintext.
	case errors.Is(decErr, crypto.ErrTOTPSecretNotEncrypted):
		// Legacy plaintext row from a pre-encryption release. Use the
		// stored value directly; arrange to migrate it on success.
		plainSecret = totpModel.Secret
		migrate = true
	default:
		// Decryption was attempted (the row IS prefixed) but failed.
		// The most likely cause is a key mismatch — operators rotating
		// --jwt-secret without re-enrolling TOTP users would lock them
		// out. Fail closed and log loudly.
		log.Error().Err(decErr).Msg("failed to decrypt stored TOTP secret; check that --jwt-secret has not changed since enrollment")
		return false, decErr
	}

	status := totp.Validate(passcode, plainSecret)
	if !status {
		// Wrong code. Don't bother with VerifiedAt or migration —
		// nothing about the row should change on a failed login.
		return false, nil
	}

	// Two reasons we may need to write the row back after a successful
	// validation:
	//   1. First-time-ever validation → record VerifiedAt
	//   2. The row is legacy plaintext → re-encrypt in place
	updateVerifiedAt := totpModel.VerifiedAt == nil
	if updateVerifiedAt {
		timeNow := time.Now().Unix()
		totpModel.VerifiedAt = &timeNow
	}

	if migrate {
		ct, encErr := crypto.EncryptTOTPSecret(plainSecret, p.deps.EncryptionKey)
		if encErr != nil {
			// Encryption failed — log and skip the migration. The
			// validation itself succeeded so we still return true; the
			// next call retries naturally because the row is unchanged.
			log.Warn().Err(encErr).Msg("totp lazy migration: encrypt failed, leaving row unchanged")
			migrate = false
		} else {
			totpModel.Secret = ct
		}
	}

	if updateVerifiedAt || migrate {
		if _, err = p.deps.StorageProvider.UpdateAuthenticator(ctx, totpModel); err != nil {
			log.Warn().Err(err).
				Bool("verified_at_update", updateVerifiedAt).
				Bool("migration_attempt", migrate).
				Msg("totp post-validate row update failed; continuing")
		} else if migrate {
			log.Info().Msg("totp lazy migration: legacy plaintext row rewritten as enc:v1:")
		}
	}

	return true, nil
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
