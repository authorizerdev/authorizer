package memorystore

import (
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/memorystore/providers"
	"github.com/authorizerdev/authorizer/server/memorystore/providers/inmemory"
	"github.com/authorizerdev/authorizer/server/memorystore/providers/redis"
)

// Provider returns the current database provider
var Provider providers.Provider

// InitMemStore initializes the memory store
func InitMemStore() error {
	var err error

	redisURL := RequiredEnvStoreObj.GetRequiredEnv().RedisURL
	if redisURL != "" {
		log.Info("Initializing Redis memory store")
		Provider, err = redis.NewRedisProvider(redisURL)
		if err != nil {
			return err
		}

		return nil
	}

	log.Info("using in memory store to save sessions")
	// if redis url is not set use in memory store
	Provider, err = inmemory.NewInMemoryProvider()
	if err != nil {
		return err
	}
	return nil
}
