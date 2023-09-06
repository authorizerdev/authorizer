package arangodb

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"image/png"
	"os"
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

func (p *provider) GenerateKeysTOTP() (*rsa.PublicKey, error) {
	key := os.Getenv("TOTP_PRIVATE_KEY")
	var privateKey *rsa.PrivateKey
	if key == "" {
		privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			return nil, err
		}

		privateKeyPEM := encodePrivateKeyToPEM(privateKey)
		os.Setenv("TOTP_PRIVATE_KEY", string(privateKeyPEM))
	}
	publicKey := privateKey.PublicKey
	return &publicKey, nil
}

func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Marshal the private key to DER format.
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	// Create a PEM block for the private key.
	privateKeyPEMBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	// Encode the PEM block to PEM format.
	privateKeyPEM := pem.EncodeToMemory(privateKeyPEMBlock)

	return privateKeyPEM
}
