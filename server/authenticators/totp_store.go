package authenticators

import (
	"github.com/authorizerdev/authorizer/server/authenticators/providers"
	"github.com/authorizerdev/authorizer/server/authenticators/providers/totp"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

// Provider is the global authenticators provider.
var Provider providers.Provider

// InitTOTPStore initializes the TOTP authenticator store if it's not disabled in the environment variables.
// It sets the global Provider variable to a new TOTP provider.
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
