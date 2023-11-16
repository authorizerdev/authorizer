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

	"github.com/authorizerdev/authorizer/server/authenticators/providers"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/refs"
)

// Generate generates a Time-Based One-Time Password (TOTP) for a user and returns the base64-encoded QR code for frontend display.
func (p *provider) Generate(ctx context.Context, id string) (*providers.AuthenticatorConfig, error) {
	var buf bytes.Buffer

	//get user details
	user, err := db.Provider.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error while getting user details")
	}

	// generate totp, Authenticators hash is valid for 30 seconds
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "authorizer",
		AccountName: refs.StringValue(user.Email),
	})
	if err != nil {
		return nil, fmt.Errorf("error while genrating totp")
	}

	//generating image for key and encoding to base64 for displaying in frontend
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("error while creating qr image for totp")
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
		return nil, fmt.Errorf("error while converting recoveryCodes to string")
	}
	recoveryCodesString := string(jsonData)

	totpModel := &models.Authenticator{
		Secret:        secret,
		RecoveryCodes: refs.NewStringRef(recoveryCodesString),
		UserID:        user.ID,
		Method:        constants.EnvKeyTOTPAuthenticator,
	}
	_, err = db.Provider.AddAuthenticator(ctx, totpModel)
	if err != nil {
		return nil, fmt.Errorf("error while inserting into totp table")
	}
	return &providers.AuthenticatorConfig{
		ScannerImage:  encodedText,
		Secret:        secret,
		RecoveryCodes: recoveryCodes,
	}, nil
}

// Validate validates a Time-Based One-Time Password (TOTP) against the stored TOTP secret for a user.
func (p *provider) Validate(ctx context.Context, passcode string, userID string) (bool, error) {
	// get totp details
	totpModel, err := db.Provider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return false, fmt.Errorf("error while getting totp details from authenticators")
	}

	status := totp.Validate(passcode, totpModel.Secret)
	// checks if user not signed in for totp and totp code is correct then VerifiedAt will be stored in db
	if totpModel.VerifiedAt == nil {
		if status {
			timeNow := time.Now().Unix()
			totpModel.VerifiedAt = &timeNow
			_, err = db.Provider.UpdateAuthenticator(ctx, totpModel)
			if err != nil {
				return false, fmt.Errorf("error while updaing authenticator table for totp")
			}
			return status, nil
		}
		return status, nil
	}
	return status, nil
}

// RecoveryCode generates a recovery code for a user's TOTP authentication, if not already verified.
func (p *provider) RecoveryCode(ctx context.Context, id string) (*string, error) {
	// get totp details
	// totpModel, err := db.Provider.GetAuthenticatorDetailsByUserId(ctx, id, constants.EnvKeyTOTPAuthenticator)
	// if err != nil {
	// 	return nil, fmt.Errorf("error while getting totp details from authenticators")
	// }
	// //TODO *totpModel.RecoveryCode == "null" used to just verify couchbase recoveryCode value to be nil
	// // have to find another way round
	// if totpModel.RecoveryCode == nil || *totpModel.RecoveryCode == "null" {
	// 	recoveryCode := utils.GenerateTOTPRecoveryCode()
	// 	totpModel.RecoveryCode = &recoveryCode

	// 	_, err = db.Provider.UpdateAuthenticator(ctx, totpModel)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("error while updaing authenticator table for totp")
	// 	}
	// 	return &recoveryCode, nil
	// }
	return nil, nil
}
