package authenticators

import (
	"github.com/authorizerdev/authorizer/server/authenticators/providers"
	"github.com/authorizerdev/authorizer/server/authenticators/providers/totp"
)

var Provider providers.Provider

func InitTOTPStore() error {
	var err error

	Provider, err = totp.NewProvider()
	if err != nil {
		return err
	}

	return nil
}
