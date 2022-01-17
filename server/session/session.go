package session

import (
	"context"
	"log"

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

// SetUserSession sets the user session in the session store
func SetUserSession(userId, accessToken, refreshToken string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.AddUserSession(userId, accessToken, refreshToken)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.AddUserSession(userId, accessToken, refreshToken)
	}
}

// DeleteUserSession deletes the particular user session from the session store
func DeleteUserSession(userId, accessToken string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.DeleteUserSession(userId, accessToken)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.DeleteUserSession(userId, accessToken)
	}
}

// DeleteAllSessions deletes all the sessions from the session store
func DeleteAllUserSession(userId string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.DeleteAllUserSession(userId)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.DeleteAllUserSession(userId)
	}
}

// GetUserSession returns the user session from the session store
func GetUserSession(userId, accessToken string) string {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		return SessionStoreObj.RedisMemoryStoreObj.GetUserSession(userId, accessToken)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		return SessionStoreObj.InMemoryStoreObj.GetUserSession(userId, accessToken)
	}

	return ""
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

// SetSocialLoginState sets the social login state in the session store
func SetSocailLoginState(key, state string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.SetSocialLoginState(key, state)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.SetSocialLoginState(key, state)
	}
}

// GetSocialLoginState returns the social login state from the session store
func GetSocailLoginState(key string) string {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		return SessionStoreObj.RedisMemoryStoreObj.GetSocialLoginState(key)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		return SessionStoreObj.InMemoryStoreObj.GetSocialLoginState(key)
	}

	return ""
}

// RemoveSocialLoginState removes the social login state from the session store
func RemoveSocialLoginState(key string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.RemoveSocialLoginState(key)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.RemoveSocialLoginState(key)
	}
}

// InitializeSessionStore initializes the SessionStoreObj based on environment variables
func InitSession() {
	if envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyRedisURL).(string) != "" {
		log.Println("using redis store to save sessions")
		opt, err := redis.ParseURL(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyRedisURL).(string))
		if err != nil {
			log.Fatalln("Error parsing redis url:", err)
		}
		rdb := redis.NewClient(opt)
		ctx := context.Background()
		_, err = rdb.Ping(ctx).Result()

		if err != nil {
			log.Fatalln("Error connecting to redis server", err)
		}
		SessionStoreObj.RedisMemoryStoreObj = &RedisStore{
			ctx:   ctx,
			store: rdb,
		}

	} else {
		log.Println("using in memory store to save sessions")
		SessionStoreObj.InMemoryStoreObj = &InMemoryStore{
			store:            map[string]map[string]string{},
			socialLoginState: map[string]string{},
		}
	}
}
