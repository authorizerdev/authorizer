package redis

import (
	"strconv"

	"github.com/authorizerdev/authorizer/server/constants"
	log "github.com/sirupsen/logrus"
)

var (
	// state store prefix
	stateStorePrefix = "authorizer_state:"
	// env store prefix
	envStorePrefix = "authorizer_env"
)

// SetUserSession sets the user session in redis store.
func (c *provider) SetUserSession(userId, key, token string) error {
	err := c.store.HSet(c.ctx, userId, key, token).Err()
	if err != nil {
		log.Debug("Error saving to redis: ", err)
		return err
	}
	return nil
}

// GetAllUserSessions returns all the user session token from the redis store.
func (c *provider) GetAllUserSessions(userID string) (map[string]string, error) {
	data, err := c.store.HGetAll(c.ctx, userID).Result()
	if err != nil {
		log.Debug("error getting all user sessions from redis store: ", err)
		return nil, err
	}

	return data, nil
}

// GetUserSession returns the user session from redis store.
func (c *provider) GetUserSession(userId, key string) (string, error) {
	var res string
	err := c.store.HGet(c.ctx, userId, key).Scan(&res)
	if err != nil {
		return "", err
	}
	return res, nil
}

// DeleteUserSession deletes the user session from redis store.
func (c *provider) DeleteUserSession(userId, key string) error {
	err := c.store.HDel(c.ctx, userId, key).Err()
	if err != nil {
		log.Debug("Error deleting user session from redis: ", err)
		return err
	}
	return nil
}

// DeleteAllUserSessions deletes all the user session from redis
func (c *provider) DeleteAllUserSessions(userID string) error {
	err := c.store.HDel(c.ctx, userID).Err()
	if err != nil {
		log.Debug("Error deleting all user sessions from redis: ", err)
		return err
	}
	return nil
}

// SetState sets the state in redis store.
func (c *provider) SetState(key, value string) error {
	err := c.store.Set(c.ctx, stateStorePrefix+key, value, 0).Err()
	if err != nil {
		log.Debug("Error saving redis token: ", err)
		return err
	}

	return nil
}

// GetState gets the state from redis store.
func (c *provider) GetState(key string) (string, error) {
	var res string
	err := c.store.Get(c.ctx, stateStorePrefix+key).Scan(&res)
	if err != nil {
		log.Debug("error getting token from redis store: ", err)
	}

	return res, err
}

// RemoveState removes the state from redis store.
func (c *provider) RemoveState(key string) error {
	err := c.store.Del(c.ctx, stateStorePrefix+key).Err()
	if err != nil {
		log.Fatalln("Error deleting redis token: ", err)
		return err
	}

	return nil
}

// UpdateEnvStore to update the whole env store object
func (c *provider) UpdateEnvStore(store map[string]interface{}) error {
	for key, value := range store {
		err := c.store.HSet(c.ctx, envStorePrefix, key, value).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetEnvStore returns the whole env store object
func (c *provider) GetEnvStore() (map[string]interface{}, error) {
	res := make(map[string]interface{})
	data, err := c.store.HGetAll(c.ctx, envStorePrefix).Result()
	if err != nil {
		return nil, err
	}
	for key, value := range data {
		if key == constants.EnvKeyDisableBasicAuthentication || key == constants.EnvKeyDisableEmailVerification || key == constants.EnvKeyDisableLoginPage || key == constants.EnvKeyDisableMagicLinkLogin || key == constants.EnvKeyDisableRedisForEnv || key == constants.EnvKeyDisableSignUp {
			boolValue, err := strconv.ParseBool(value)
			if err != nil {
				return res, err
			}
			res[key] = boolValue
		} else {
			res[key] = value
		}
	}
	return res, nil
}

// UpdateEnvVariable to update the particular env variable
func (c *provider) UpdateEnvVariable(key string, value interface{}) error {
	err := c.store.HSet(c.ctx, envStorePrefix, key, value).Err()
	if err != nil {
		log.Debug("Error saving redis token: ", err)
		return err
	}
	return nil
}

// GetStringStoreEnvVariable to get the string env variable from env store
func (c *provider) GetStringStoreEnvVariable(key string) (string, error) {
	var res string
	err := c.store.HGet(c.ctx, envStorePrefix, key).Scan(&res)
	if err != nil {
		return "", nil
	}

	return res, nil
}

// GetBoolStoreEnvVariable to get the bool env variable from env store
func (c *provider) GetBoolStoreEnvVariable(key string) (bool, error) {
	var res bool
	err := c.store.HGet(c.ctx, envStorePrefix, key).Scan(res)
	if err != nil {
		return false, nil
	}

	return res, nil
}
