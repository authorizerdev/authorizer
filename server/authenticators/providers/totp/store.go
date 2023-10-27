package totp

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/utils"
)

// Generate generates a Time-Based One-Time Password (TOTP) for a user and returns the base64-encoded QR code for frontend display.
func (p *provider) Generate(ctx context.Context, id string) (*string, error) {
	var buf bytes.Buffer
	var totpModel models.Authenticators

	//get user details
	user, err := db.Provider.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error while getting user details")
	}

	// generate totp, Authenticators hash is valid for 30 seconds
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "authorizer",
		AccountName: *user.Email,
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
	totpModel.Secret = secret
	totpModel.UserID = user.ID
	totpModel.Method = constants.EnvKeyTOTPAuthenticator
	_, err = db.Provider.AddAuthenticator(ctx, totpModel)
	if err != nil {
		return nil, fmt.Errorf("error while inserting into totp table")
	}
	return &encodedText, nil
}

// Validate validates a Time-Based One-Time Password (TOTP) against the stored TOTP secret for a user.
func (p *provider) Validate(ctx context.Context, passcode string, id string) (bool, error) {
	// get totp details
	totpModel, err := db.Provider.GetAuthenticatorDetailsByUserId(ctx, id, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return false, fmt.Errorf("error while getting totp details from authenticators")
	}

	status := totp.Validate(passcode, totpModel.Secret)
	// checks if user not signed in for totp and totp code is correct then VerifiedAt will be stored in db
	if totpModel.VerifiedAt == nil {
		if status {
			timeNow := time.Now().Unix()
			totpModel.VerifiedAt = &timeNow
			_, err = db.Provider.UpdateAuthenticator(ctx, *totpModel)
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
	totpModel, err := db.Provider.GetAuthenticatorDetailsByUserId(ctx, id, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return nil, fmt.Errorf("error while getting totp details from authenticators")
	}
	//TODO *totpModel.RecoveryCode == "null" used to just verify couchbase recoveryCode value to be nil
	// have to find another way round
	if totpModel.RecoveryCode == nil || *totpModel.RecoveryCode == "null" {
		recoveryCode := utils.GenerateTOTPRecoveryCode()
		totpModel.RecoveryCode = &recoveryCode

		_, err = db.Provider.UpdateAuthenticator(ctx, *totpModel)
		if err != nil {
			return nil, fmt.Errorf("error while updaing authenticator table for totp")
		}
		return &recoveryCode, nil
	}
	return nil, nil
}
