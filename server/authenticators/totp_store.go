package authenticators

import (
	"github.com/authorizerdev/authorizer/server/authenticators/providers"
	"github.com/authorizerdev/authorizer/server/authenticators/providers/totp"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

var Provider providers.Provider

func InitTOTPStore() error {
	var err error
	isTOTPEnvServiceDisabled, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableTOTPLogin)

	if !isTOTPEnvServiceDisabled {
		Provider, err = totp.NewProvider()
		if err != nil {
			return err
		}
	}
	return nil
}
