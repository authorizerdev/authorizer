package redis

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	// session store prefix
	sessionStorePrefix = "authorizer_session_"
	// env store prefix
	envStorePrefix = "authorizer_env_"
)

// ClearStore clears the redis store for authorizer related tokens
func (c *provider) ClearStore() error {
	err := c.store.Del(c.ctx, sessionStorePrefix+"*").Err()
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
	err := c.store.Set(c.ctx, sessionStorePrefix+key, value, 0).Err()
	if err != nil {
		log.Debug("Error saving redis token: ", err)
		return err
	}

	return nil
}

// GetState gets the state from redis store.
func (c *provider) GetState(key string) (string, error) {
	var res string
	err := c.store.Get(c.ctx, sessionStorePrefix+key).Scan(&res)
	if err != nil {
		log.Debug("error getting token from redis store: ", err)
	}

	return res, err
}

// RemoveState removes the state from redis store.
func (c *provider) RemoveState(key string) error {
	err := c.store.Del(c.ctx, sessionStorePrefix+key).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token: ", err)
		return err
	}

	return nil
}

// UpdateEnvStore to update the whole env store object
func (c *provider) UpdateEnvStore(store map[string]interface{}) error {
	for key, value := range store {
		err := c.store.Set(c.ctx, envStorePrefix+key, value, 0).Err()
		if err != nil {
			log.Debug("Error saving redis token: ", err)
			return err
		}
	}
	return nil
}

// GetEnvStore returns the whole env store object
func (c *provider) GetEnvStore() (map[string]interface{}, error) {
	var res map[string]interface{}
	err := c.store.HGetAll(c.ctx, envStorePrefix+"*").Scan(res)
	if err != nil {
		log.Debug("error getting token from redis store: ", err)
		return nil, err
	}

	return res, nil
}

// UpdateEnvVariable to update the particular env variable
func (c *provider) UpdateEnvVariable(key string, value interface{}) error {
	err := c.store.Set(c.ctx, envStorePrefix+key, value, 0).Err()
	if err != nil {
		log.Debug("Error saving redis token: ", err)
		return err
	}
	return nil
}

// GetStringStoreEnvVariable to get the string env variable from env store
func (c *provider) GetStringStoreEnvVariable(key string) (string, error) {
	var res string
	err := c.store.Get(c.ctx, envStorePrefix+key).Scan(&res)
	if err != nil {
		log.Debug("error getting token from redis store: ", err)
		return "", err
	}

	return res, nil
}

// GetBoolStoreEnvVariable to get the bool env variable from env store
func (c *provider) GetBoolStoreEnvVariable(key string) (bool, error) {
	var res bool
	err := c.store.Get(c.ctx, envStorePrefix+key).Scan(res)
	if err != nil {
		log.Debug("error getting token from redis store: ", err)
		return false, err
	}

	return res, nil
}
