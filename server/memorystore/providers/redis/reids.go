package redis

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

// ClearStore clears the redis store for authorizer related tokens
func (c *provider) ClearStore() error {
	err := c.store.Del(c.ctx, "authorizer_*").Err()
	if err != nil {
		log.Debug("Error clearing redis store: ", err)
		return err
	}

	return nil
}

// GetUserSessions returns all the user session token from the redis store.
func (c *provider) GetUserSessions(userID string) map[string]string {
	data, err := c.store.HGetAll(c.ctx, "*").Result()
	if err != nil {
		log.Debug("error getting token from redis store: ", err)
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
func (c *provider) DeleteAllUserSession(userId string) error {
	sessions := c.GetUserSessions(userId)
	for k, v := range sessions {
		if k == "token" {
			err := c.store.Del(c.ctx, v).Err()
			if err != nil {
				log.Debug("Error deleting redis token: ", err)
				return err
			}
		}
	}

	return nil
}

// SetState sets the state in redis store.
func (c *provider) SetState(key, value string) error {
	err := c.store.Set(c.ctx, "authorizer_"+key, value, 0).Err()
	if err != nil {
		log.Debug("Error saving redis token: ", err)
		return err
	}

	return nil
}

// GetState gets the state from redis store.
func (c *provider) GetState(key string) string {
	state := ""
	state, err := c.store.Get(c.ctx, "authorizer_"+key).Result()
	if err != nil {
		log.Debug("error getting token from redis store: ", err)
	}

	return state
}

// RemoveState removes the state from redis store.
func (c *provider) RemoveState(key string) error {
	err := c.store.Del(c.ctx, "authorizer_"+key).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token: ", err)
		return err
	}

	return nil
}
