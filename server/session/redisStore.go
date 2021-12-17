package session

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	ctx   context.Context
	store *redis.Client
}

func (c *RedisStore) AddToken(userId, accessToken, refreshToken string) {
	err := c.store.HMSet(c.ctx, "authorizer_"+userId, map[string]string{
		accessToken: refreshToken,
	}).Err()
	if err != nil {
		log.Fatalln("Error saving redis token:", err)
	}
}

func (c *RedisStore) DeleteUserSession(userId string) {
	err := c.store.Del(c.ctx, "authorizer_"+userId).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}

func (c *RedisStore) DeleteVerificationRequest(userId, accessToken string) {
	err := c.store.HDel(c.ctx, "authorizer_"+userId, accessToken).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}

func (c *RedisStore) ClearStore() {
	err := c.store.Del(c.ctx, "authorizer_*").Err()
	if err != nil {
		log.Fatalln("Error clearing redis store:", err)
	}
}

func (c *RedisStore) GetToken(userId, accessToken string) string {
	token := ""
	res, err := c.store.HMGet(c.ctx, "authorizer_"+userId, accessToken).Result()
	if err != nil {
		log.Println("Error getting token from redis store:", err)
	}
	if len(res) > 0 && res[0] != nil {
		token = fmt.Sprintf("%v", res[0])
	}
	return token
}

func (c *RedisStore) SetSocialLoginState(key, state string) {
	err := c.store.Set(c.ctx, key, state, 0).Err()
	if err != nil {
		log.Fatalln("Error saving redis token:", err)
	}
}

func (c *RedisStore) GetSocialLoginState(key string) string {
	state := ""
	state, err := c.store.Get(c.ctx, key).Result()
	if err != nil {
		log.Println("Error getting token from redis store:", err)
	}

	return state
}

func (c *RedisStore) RemoveSocialLoginState(key string) {
	err := c.store.Del(c.ctx, key).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}
