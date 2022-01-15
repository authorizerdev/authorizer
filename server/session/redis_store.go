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

// AddUserSession adds the user session to redis
func (c *RedisStore) AddUserSession(userId, accessToken, refreshToken string) {
	err := c.store.HMSet(c.ctx, "authorizer_"+userId, map[string]string{
		accessToken: refreshToken,
	}).Err()
	if err != nil {
		log.Fatalln("Error saving redis token:", err)
	}
}

// DeleteAllUserSession deletes all the user session from redis
func (c *RedisStore) DeleteAllUserSession(userId string) {
	err := c.store.Del(c.ctx, "authorizer_"+userId).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}

// DeleteUserSession deletes the particular user session from redis
func (c *RedisStore) DeleteUserSession(userId, accessToken string) {
	err := c.store.HDel(c.ctx, "authorizer_"+userId, accessToken).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}

// ClearStore clears the redis store for authorizer related tokens
func (c *RedisStore) ClearStore() {
	err := c.store.Del(c.ctx, "authorizer_*").Err()
	if err != nil {
		log.Fatalln("Error clearing redis store:", err)
	}
}

// GetUserSession returns the user session token from the redis store.
func (c *RedisStore) GetUserSession(userId, accessToken string) string {
	token := ""
	res, err := c.store.HMGet(c.ctx, "authorizer_"+userId, accessToken).Result()
	if err != nil {
		log.Println("error getting token from redis store:", err)
	}
	if len(res) > 0 && res[0] != nil {
		token = fmt.Sprintf("%v", res[0])
	}
	return token
}

// SetSocialLoginState sets the social login state in redis store.
func (c *RedisStore) SetSocialLoginState(key, state string) {
	err := c.store.Set(c.ctx, key, state, 0).Err()
	if err != nil {
		log.Fatalln("Error saving redis token:", err)
	}
}

// GetSocialLoginState gets the social login state from redis store.
func (c *RedisStore) GetSocialLoginState(key string) string {
	state := ""
	state, err := c.store.Get(c.ctx, key).Result()
	if err != nil {
		log.Println("error getting token from redis store:", err)
	}

	return state
}

// RemoveSocialLoginState removes the social login state from redis store.
func (c *RedisStore) RemoveSocialLoginState(key string) {
	err := c.store.Del(c.ctx, key).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}
