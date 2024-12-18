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
	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// Generate generates a Time-Based One-Time Password (TOTP) for a user and returns the base64-encoded QR code for frontend display.
func (p *provider) Generate(ctx context.Context, id string) (*config.AuthenticatorConfig, error) {
	log := p.deps.Log.With().Str("func", "Generate (totp provider)").Logger()
	var buf bytes.Buffer
	// Get user details
	user, err := p.deps.DB.GetUserByID(ctx, id)
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
	totpModel := &schemas.Authenticator{
		Secret:        secret,
		RecoveryCodes: refs.NewStringRef(recoveryCodesString),
		UserID:        user.ID,
		Method:        constants.EnvKeyTOTPAuthenticator,
	}
	authenticator, err := p.deps.DB.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		log.Debug().Err(err).Msg("error getting authenticator details")
		// continue
	}
	if authenticator == nil {
		// if authenticator is nil then create new authenticator
		_, err = p.deps.DB.AddAuthenticator(ctx, totpModel)
		if err != nil {
			return nil, err
		}
	} else {
		authenticator.Secret = secret
		authenticator.RecoveryCodes = refs.NewStringRef(recoveryCodesString)
		// if authenticator is not nil then update authenticator
		_, err = p.deps.DB.UpdateAuthenticator(ctx, authenticator)
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
	totpModel, err := p.deps.DB.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return false, err
	}
	// validate totp
	status := totp.Validate(passcode, totpModel.Secret)
	// checks if user not signed in for totp and totp code is correct then VerifiedAt will be stored in db
	if totpModel.VerifiedAt == nil && status {
		timeNow := time.Now().Unix()
		totpModel.VerifiedAt = &timeNow
		_, err = p.deps.DB.UpdateAuthenticator(ctx, totpModel)
		if err != nil {
			return false, err
		}
	}
	return status, nil
}

// ValidateRecoveryCode validates a Time-Based One-Time Password (TOTP) recovery code against the stored TOTP recovery code for a user.
func (p *provider) ValidateRecoveryCode(ctx context.Context, recoveryCode, userID string) (bool, error) {
	// get totp details
	totpModel, err := p.deps.DB.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
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
	_, err = p.deps.DB.UpdateAuthenticator(ctx, totpModel)
	if err != nil {
		return false, err
	}
	return true, nil
}
