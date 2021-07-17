package session

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/yauthdev/yauth/server/constants"
)

type SessionStore struct {
	InMemoryStoreObj    *InMemoryStore
	RedisMemoryStoreObj *RedisStore
}

var SessionStoreObj SessionStore

func SetToken(userId, token string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.AddToken(userId, token)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.AddToken(userId, token)
	}
}

func DeleteToken(userId string) {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		SessionStoreObj.RedisMemoryStoreObj.DeleteToken(userId)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		SessionStoreObj.InMemoryStoreObj.DeleteToken(userId)
	}
}

func GetToken(userId string) string {
	if SessionStoreObj.RedisMemoryStoreObj != nil {
		return SessionStoreObj.RedisMemoryStoreObj.GetToken(userId)
	}
	if SessionStoreObj.InMemoryStoreObj != nil {
		return SessionStoreObj.InMemoryStoreObj.GetToken(userId)
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

func init() {
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
			store: make(map[string]string),
		}
	}
}
