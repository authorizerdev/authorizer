package session

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/yauthdev/yauth/server/constants"
)

type SessionStore struct {
	inMemoryStoreObj    *InMemoryStore
	redisMemoryStoreObj *RedisStore
}

var SessionStoreObj SessionStore

func setToken(userId, token string) {
	if SessionStoreObj.redisMemoryStoreObj != nil {
		SessionStoreObj.redisMemoryStoreObj.AddToken(userId, token)
	}
	if SessionStoreObj.inMemoryStoreObj != nil {
		SessionStoreObj.inMemoryStoreObj.AddToken(userId, token)
	}
}

func deleteToken(userId string) {
	if SessionStoreObj.redisMemoryStoreObj != nil {
		SessionStoreObj.redisMemoryStoreObj.DeleteToken(userId)
	}
	if SessionStoreObj.inMemoryStoreObj != nil {
		SessionStoreObj.inMemoryStoreObj.DeleteToken(userId)
	}
}

func getToken(userId string) {
	if SessionStoreObj.redisMemoryStoreObj != nil {
		SessionStoreObj.redisMemoryStoreObj.GetToken(userId)
	}
	if SessionStoreObj.inMemoryStoreObj != nil {
		SessionStoreObj.inMemoryStoreObj.GetToken(userId)
	}
}

func clearStore() {
	if SessionStoreObj.redisMemoryStoreObj != nil {
		SessionStoreObj.redisMemoryStoreObj.ClearStore()
	}
	if SessionStoreObj.inMemoryStoreObj != nil {
		SessionStoreObj.inMemoryStoreObj.ClearStore()
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
		SessionStoreObj.redisMemoryStoreObj = &RedisStore{
			ctx:   ctx,
			store: rdb,
		}

	} else {
		log.Println("Using in memory store to save sessions")
		SessionStoreObj.inMemoryStoreObj = &InMemoryStore{
			store: make(map[string]string),
		}
	}
}
