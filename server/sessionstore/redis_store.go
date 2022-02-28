package sessionstore

import (
	"context"
	"fmt"
	"log"
)

type RedisStore struct {
	ctx   context.Context
	store RedisSessionClient
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

// GetUserSessions returns all the user session token from the redis store.
func (c *RedisStore) GetUserSessions(userID string) map[string]string {
	res, err := c.store.HGetAll(c.ctx, "authorizer_"+userID).Result()
	if err != nil {
		log.Println("error getting token from redis store:", err)
	}

	return res
}

// SetState sets the state in redis store.
func (c *RedisStore) SetState(key, state string) {
	err := c.store.Set(c.ctx, key, state, 0).Err()
	if err != nil {
		log.Fatalln("Error saving redis token:", err)
	}
}

// GetState gets the state from redis store.
func (c *RedisStore) GetState(key string) string {
	state := ""
	state, err := c.store.Get(c.ctx, key).Result()
	if err != nil {
		log.Println("error getting token from redis store:", err)
	}

	return state
}

// RemoveState removes the state from redis store.
func (c *RedisStore) RemoveState(key string) {
	err := c.store.Del(c.ctx, key).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}
