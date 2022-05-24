package sessionstore

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
)

type RedisStore struct {
	ctx   context.Context
	store RedisSessionClient
}

// ClearStore clears the redis store for authorizer related tokens
func (c *RedisStore) ClearStore() {
	err := c.store.Del(c.ctx, "authorizer_*").Err()
	if err != nil {
		log.Debug("Error clearing redis store:", err)
	}
}

// GetUserSessions returns all the user session token from the redis store.
func (c *RedisStore) GetUserSessions(userID string) map[string]string {
	data, err := c.store.HGetAll(c.ctx, "*").Result()
	if err != nil {
		log.Debug("error getting token from redis store:", err)
	}

	res := map[string]string{}
	for k, v := range data {
		split := strings.Split(v, "@")
		if split[1] == userID {
			res[k] = split[0]
		}
	}

	return res
}

// DeleteAllUserSession deletes all the user session from redis
func (c *RedisStore) DeleteAllUserSession(userId string) {
	sessions := GetUserSessions(userId)
	for k, v := range sessions {
		if k == "token" {
			err := c.store.Del(c.ctx, v)
			if err != nil {
				log.Debug("Error deleting redis token:", err)
			}
		}
	}
}

// SetState sets the state in redis store.
func (c *RedisStore) SetState(key, value string) {
	err := c.store.Set(c.ctx, "authorizer_"+key, value, 0).Err()
	if err != nil {
		log.Debug("Error saving redis token:", err)
	}
}

// GetState gets the state from redis store.
func (c *RedisStore) GetState(key string) string {
	state := ""
	state, err := c.store.Get(c.ctx, "authorizer_"+key).Result()
	if err != nil {
		log.Debug("error getting token from redis store:", err)
	}

	return state
}

// RemoveState removes the state from redis store.
func (c *RedisStore) RemoveState(key string) {
	err := c.store.Del(c.ctx, "authorizer_"+key).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token:", err)
	}
}
