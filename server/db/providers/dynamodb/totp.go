package dynamodb

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/authorizerdev/authorizer/server/crypto"
)

func (p *provider) GenerateTotp(ctx context.Context, id string) (*string, error) {
	var buf bytes.Buffer
	//get user details
	user, err := p.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error while getting user details")
	}

	// generate totp, TOTP hash is valid for 30 seconds
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "authorizer",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("error while genrating totp")
	}

	// get secret for user
	secret := key.Secret()

	//generating image for key and encoding to base64 for displaying in frontend
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("error while creating qr image for totp")
	}
	png.Encode(&buf, img)
	encodedText := crypto.EncryptB64(buf.String())

	// update user totp secret in db
	user.UpdatedAt = time.Now().Unix()
	user.TotpSecret = &secret
	_, err = p.UpdateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("error while updating user's totp secret")
	}

	return &encodedText, nil
}

func (p *provider) ValidatePasscode(ctx context.Context, passcode string, id string) (bool, error) {
	// get user details
	user, err := p.GetUserByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("error while getting user details")
	}

	// validate passcode inputted by user
	for {
		status := totp.Validate(passcode, *user.TotpSecret)
		if status {
			return status, nil
		}
	}
}
