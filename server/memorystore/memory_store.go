package memorystore

import (
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
		constants.EnvKeyDisableBasicAuthentication: false,
		constants.EnvKeyDisableMagicLinkLogin:      false,
		constants.EnvKeyDisableEmailVerification:   false,
		constants.EnvKeyDisableLoginPage:           false,
		constants.EnvKeyDisableSignUp:              false,
	}

	redisURL := RequiredEnvStoreObj.GetRequiredEnv().RedisURL
	if redisURL != "" {
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
