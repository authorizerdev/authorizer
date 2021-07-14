package session

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	ctx   context.Context
	store *redis.Client
}

func (c *RedisStore) AddToken(userId, token string) {
	err := c.store.Set(c.ctx, "yauth_"+userId, token, 0).Err()
	if err != nil {
		log.Fatalln("Error saving redis token:", err)
	}
}

func (c *RedisStore) DeleteToken(userId string) {
	err := c.store.Del(c.ctx, "yauth_"+userId).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}

func (c *RedisStore) ClearStore() {
	err := c.store.Del(c.ctx, "yauth_*").Err()
	if err != nil {
		log.Fatalln("Error clearing redis store:", err)
	}
}

func (c *RedisStore) GetToken(userId string) string {
	token := ""
	token, err := c.store.Get(c.ctx, "yauth_"+userId).Result()
	if err != nil {
		log.Println("Error getting token from redis store:", err)
	}
	return token
}
