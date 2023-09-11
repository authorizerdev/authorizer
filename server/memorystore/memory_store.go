package memorystore

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore/providers"
	"github.com/authorizerdev/authorizer/server/memorystore/providers/inmemory"
	"github.com/authorizerdev/authorizer/server/memorystore/providers/redis"
)

// Provider returns the current database provider
var Provider providers.Provider

// InitMemStore initializes the memory store
func InitMemStore() error {
	var err error

	defaultEnvs := map[string]interface{}{
		// string envs
		constants.EnvKeyJwtRoleClaim:     "role",
		constants.EnvKeyOrganizationName: "Authorizer",
		constants.EnvKeyOrganizationLogo: "https://www.authorizer.dev/images/logo.png",

		// boolean envs
		constants.EnvKeyDisableBasicAuthentication:       false,
		constants.EnvKeyDisableMobileBasicAuthentication: false,
		constants.EnvKeyDisableMagicLinkLogin:            false,
		constants.EnvKeyDisableEmailVerification:         false,
		constants.EnvKeyDisableLoginPage:                 false,
		constants.EnvKeyDisableSignUp:                    false,
		constants.EnvKeyDisableStrongPassword:            false,
		constants.EnvKeyIsEmailServiceEnabled:            false,
		constants.EnvKeyIsSMSServiceEnabled:              false,
		constants.EnvKeyEnforceMultiFactorAuthentication: false,
		constants.EnvKeyDisableMultiFactorAuthentication: false,
		constants.EnvKeyAppCookieSecure:                  true,
		constants.EnvKeyAdminCookieSecure:                true,
		constants.EnvKeyDisablePlayGround:                true,
	}

	requiredEnvs := RequiredEnvStoreObj.GetRequiredEnv()
	requiredEnvMap := make(map[string]interface{})
	requiredEnvBytes, err := json.Marshal(requiredEnvs)
	if err != nil {
		log.Debug("Error while marshalling required envs: ", err)
		return err
	}
	err = json.Unmarshal(requiredEnvBytes, &requiredEnvMap)
	if err != nil {
		log.Debug("Error while unmarshalling required envs: ", err)
		return err
	}

	// merge default envs with required envs
	for key, val := range requiredEnvMap {
		defaultEnvs[key] = val
	}

	redisURL := requiredEnvs.RedisURL
	if redisURL != "" && !requiredEnvs.DisableRedisForEnv {
		log.Info("Initializing Redis memory store")
		Provider, err = redis.NewRedisProvider(redisURL)
		if err != nil {
			return err
		}

		// set default envs in redis
		Provider.UpdateEnvStore(defaultEnvs)

		return nil
	}

	log.Info("using in memory store to save sessions")
	// if redis url is not set use in memory store
	Provider, err = inmemory.NewInMemoryProvider()
	if err != nil {
		return err
	}
	// set default envs in local env
	Provider.UpdateEnvStore(defaultEnvs)
	return nil
}
