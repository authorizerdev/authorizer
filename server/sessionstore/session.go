package sessionstore

import (
	"context"
	"log"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/go-redis/redis/v8"
)

// SessionStore is a struct that defines available session stores
// If redis store is available, higher preference is given to that store.
// Else in memory store is used.
type SessionStore struct {
	InMemoryStoreObj    *InMemoryStore
	RedisMemoryStoreObj *RedisStore
}

// SessionStoreObj is a global variable that holds the
// reference to various session store instances
var SessionStoreObj SessionStore

// DeleteAllSessions deletes all the sessions from the session store
func DeleteAllUserSession(userId string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.DeleteAllUserSession(userId)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.DeleteAllUserSession(userId)
	}
}

// GetUserSessions returns all the user sessions from the session store
func GetUserSessions(userId string) map[string]string {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		return SessionStoreObj.RedisMemoryStoreObj.GetUserSessions(userId)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		return SessionStoreObj.InMemoryStoreObj.GetUserSessions(userId)
	}

	return nil
}

// ClearStore clears the session store for authorizer tokens
func ClearStore() {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.ClearStore()
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.ClearStore()
	}
}

// SetState sets the login state (key, value form) in the session store
func SetState(key, state string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.SetState(key, state)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.SetState(key, state)
	}
}

// GetState returns the state from the session store
func GetState(key string) string {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		return SessionStoreObj.RedisMemoryStoreObj.GetState(key)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		return SessionStoreObj.InMemoryStoreObj.GetState(key)
	}

	return ""
}

// RemoveState removes the social login state from the session store
func RemoveState(key string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.RemoveState(key)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.RemoveState(key)
	}
}

// InitializeSessionStore initializes the SessionStoreObj based on environment variables
func InitSession() error {
	if envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyRedisURL) != "" {
		log.Println("using redis store to save sessions")

		redisURL := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyRedisURL)
		redisURLHostPortsList := strings.Split(redisURL, ",")

		if len(redisURLHostPortsList) > 1 {
			opt, err := redis.ParseURL(redisURLHostPortsList[0])
			if err != nil {
				return err
			}
			urls := []string{opt.Addr}
			urlList := redisURLHostPortsList[1:]
			urls = append(urls, urlList...)
			clusterOpt := &redis.ClusterOptions{Addrs: urls}

			rdb := redis.NewClusterClient(clusterOpt)
			ctx := context.Background()
			_, err = rdb.Ping(ctx).Result()
			if err != nil {
				return err
			}
			SessionStoreObj.RedisMemoryStoreObj = &RedisStore{
				ctx:   ctx,
				store: rdb,
			}

			// return on successful initialization
			return nil
		}

		opt, err := redis.ParseURL(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyRedisURL))
		if err != nil {
			return err
		}

		rdb := redis.NewClient(opt)
		ctx := context.Background()
		_, err = rdb.Ping(ctx).Result()
		if err != nil {
			return err
		}

		SessionStoreObj.RedisMemoryStoreObj = &RedisStore{
			ctx:   ctx,
			store: rdb,
		}

		// return on successful initialization
		return nil
	}

	// if redis url is not set use in memory store
	SessionStoreObj.InMemoryStoreObj = &InMemoryStore{
		sessionStore: map[string]map[string]string{},
		stateStore:   map[string]string{},
	}

	return nil
}
