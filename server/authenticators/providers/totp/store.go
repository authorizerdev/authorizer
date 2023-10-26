package totp

import (
	"bytes"
	"context"
	"fmt"
	"github.com/authorizerdev/authorizer/server/utils"
	"image/png"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
)

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

func (p *provider) Validate(ctx context.Context, passcode string, id string) (bool, error) {
	// get totp details
	totpModel, err := db.Provider.GetAuthenticatorDetailsByUserId(ctx, id, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return false, fmt.Errorf("error while getting totp details from authenticators")
	}

	status := totp.Validate(passcode, totpModel.Secret)
	if totpModel.VerifiedAt == nil {
		if status {
			timeNow := time.Now().Unix()
			totpModel.VerifiedAt = &timeNow
			fmt.Printf("totpModel validate %+v \n", totpModel)
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

func (p *provider) RecoveryCode(ctx context.Context, id string) (*string, error) {
	// get totp details
	totpModel, err := db.Provider.GetAuthenticatorDetailsByUserId(ctx, id, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return nil, fmt.Errorf("error while getting totp details from authenticators")
	}

	if totpModel.VerifiedAt == nil {
		recoveryCode := utils.GenerateTOTPRecoveryCode()
		timeNow := time.Now().Unix()

		totpModel.VerifiedAt = &timeNow
		totpModel.RecoveryCode = &recoveryCode

		_, err = db.Provider.UpdateAuthenticator(ctx, *totpModel)
		if err != nil {
			return nil, fmt.Errorf("error while updaing authenticator table for totp")
		}
		return &recoveryCode, nil
	}
	return nil, nil
}
