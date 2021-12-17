package session

import (
	"context"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/go-redis/redis/v8"
)

type SessionStore struct {
	InMemoryStoreObj    *InMemoryStore
	RedisMemoryStoreObj *RedisStore
}

var SessionStoreObj SessionStore

func SetToken(userId, accessToken, refreshToken string) {
	// TODO: Set session information in db for all the sessions that gets generated
	// it should async go function

	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.AddToken(userId, accessToken, refreshToken)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.AddToken(userId, accessToken, refreshToken)
	}
}

func DeleteVerificationRequest(userId, accessToken string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.DeleteVerificationRequest(userId, accessToken)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.DeleteVerificationRequest(userId, accessToken)
	}
}

func DeleteUserSession(userId string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.DeleteUserSession(userId)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.DeleteUserSession(userId)
	}
}

func GetToken(userId, accessToken string) string {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		return SessionStoreObj.RedisMemoryStoreObj.GetToken(userId, accessToken)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		return SessionStoreObj.InMemoryStoreObj.GetToken(userId, accessToken)
	}

	return ""
}

func ClearStore() {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.ClearStore()
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.ClearStore()
	}
}

func SetSocailLoginState(key, state string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.SetSocialLoginState(key, state)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.SetSocialLoginState(key, state)
	}
}

func GetSocailLoginState(key string) string {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		return SessionStoreObj.RedisMemoryStoreObj.GetSocialLoginState(key)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		return SessionStoreObj.InMemoryStoreObj.GetSocialLoginState(key)
	}

	return ""
}

func RemoveSocialLoginState(key string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.RemoveSocialLoginState(key)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.RemoveSocialLoginState(key)
	}
}

func InitSession() {
	if constants.REDIS_URL != "" {
		log.Println("Using redis store to save sessions")
		opt, err := redis.ParseURL(constants.REDIS_URL)
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
		log.Println("Using in memory store to save sessions")
		SessionStoreObj.InMemoryStoreObj = &InMemoryStore{
			store:            map[string]map[string]string{},
			socialLoginState: map[string]string{},
		}
	}
}
