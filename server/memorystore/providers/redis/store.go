package redis

import (
	"fmt"
	"strconv"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	log "github.com/sirupsen/logrus"
)

var (
	// state store prefix
	stateStorePrefix = "authorizer_state:"
	// env store prefix
	envStorePrefix = "authorizer_env"
)

const mfaSessionPrefix = "mfa_sess_"

// SetUserSession sets the user session for given user identifier in form recipe:user_id
func (c *provider) SetUserSession(userId, key, token string, expiration int64) error {
	currentTime := time.Now()
	expireTime := time.Unix(expiration, 0)
	duration := expireTime.Sub(currentTime)
	err := c.store.Set(c.ctx, fmt.Sprintf("%s:%s", userId, key), token, duration).Err()
	if err != nil {
		log.Debug("Error saving user session to redis: ", err)
		return err
	}
	return nil
}

// GetUserSession returns the user session from redis store.
func (c *provider) GetUserSession(userId, key string) (string, error) {
	data, err := c.store.Get(c.ctx, fmt.Sprintf("%s:%s", userId, key)).Result()
	if err != nil {
		return "", err
	}
	return data, nil
}

// DeleteUserSession deletes the user session from redis store.
func (c *provider) DeleteUserSession(userId, key string) error {
	if err := c.store.Del(c.ctx, fmt.Sprintf("%s:%s", userId, constants.TokenTypeSessionToken+"_"+key)).Err(); err != nil {
		log.Debug("Error deleting user session from redis: ", err)
		// continue
	}
	if err := c.store.Del(c.ctx, fmt.Sprintf("%s:%s", userId, constants.TokenTypeAccessToken+"_"+key)).Err(); err != nil {
		log.Debug("Error deleting user session from redis: ", err)
		// continue
	}
	if err := c.store.Del(c.ctx, fmt.Sprintf("%s:%s", userId, constants.TokenTypeRefreshToken+"_"+key)).Err(); err != nil {
		log.Debug("Error deleting user session from redis: ", err)
		// continue
	}
	return nil
}

// DeleteAllUserSessions deletes all the user session from redis
func (c *provider) DeleteAllUserSessions(userID string) error {
	res := c.store.Keys(c.ctx, fmt.Sprintf("*%s*", userID))
	if res.Err() != nil {
		log.Debug("Error getting all user sessions from redis: ", res.Err())
		return res.Err()
	}
	keys := res.Val()
	for _, key := range keys {
		err := c.store.Del(c.ctx, key).Err()
		if err != nil {
			log.Debug("Error deleting all user sessions from redis: ", err)
			continue
		}
	}
	return nil
}

// DeleteSessionForNamespace to delete session for a given namespace example google,github
func (c *provider) DeleteSessionForNamespace(namespace string) error {
	res := c.store.Keys(c.ctx, fmt.Sprintf("%s:*", namespace))
	if res.Err() != nil {
		log.Debug("Error getting all user sessions from redis: ", res.Err())
		return res.Err()
	}
	keys := res.Val()
	for _, key := range keys {
		err := c.store.Del(c.ctx, key).Err()
		if err != nil {
			log.Debug("Error deleting all user sessions from redis: ", err)
			continue
		}
	}
	return nil
}

// SetMfaSession sets the mfa session with key and value of email
func (c *provider) SetMfaSession(email, key string, expiration int64) error {
	currentTime := time.Now()
	expireTime := time.Unix(expiration, 0)
	duration := expireTime.Sub(currentTime)
	err := c.store.Set(c.ctx, fmt.Sprintf("%s%s:%s", mfaSessionPrefix, email, key), email, duration).Err()
	if err != nil {
		log.Debug("Error saving user session to redis: ", err)
		return err
	}
	return nil
}

	// GetMfaSession returns value of given mfa session
func (c *provider) GetMfaSession(email, key string) (string, error) {
	data, err := c.store.Get(c.ctx, fmt.Sprintf("%s%s:%s", mfaSessionPrefix, email, key)).Result()
	if err != nil {
		return "", err
	}
	return data, nil
}

// DeleteMfaSession deletes given mfa session from in-memory store.
func (c *provider) DeleteMfaSession(email, key string) error {
	if err := c.store.Del(c.ctx, fmt.Sprintf("%s%s:%s", mfaSessionPrefix, email, key)).Err(); err != nil {
		log.Debug("Error deleting user session from redis: ", err)
		// continue
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
	data, err := c.store.Get(c.ctx, stateStorePrefix+key).Result()
	if err != nil {
		log.Debug("error getting token from redis store: ", err)
		return "", err
	}

	return data, err
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
		if key == constants.EnvKeyDisableBasicAuthentication || key == constants.EnvKeyDisableMobileBasicAuthentication || key == constants.EnvKeyDisableEmailVerification || key == constants.EnvKeyDisableLoginPage || key == constants.EnvKeyDisableMagicLinkLogin || key == constants.EnvKeyDisableRedisForEnv || key == constants.EnvKeyDisableSignUp || key == constants.EnvKeyDisableStrongPassword || key == constants.EnvKeyIsEmailServiceEnabled || key == constants.EnvKeyIsSMSServiceEnabled || key == constants.EnvKeyEnforceMultiFactorAuthentication || key == constants.EnvKeyDisableMultiFactorAuthentication || key == constants.EnvKeyAppCookieSecure || key == constants.EnvKeyAdminCookieSecure {
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
	data, err := c.store.HGet(c.ctx, envStorePrefix, key).Result()
	if err != nil {
		return "", nil
	}

	return data, nil
}

// GetBoolStoreEnvVariable to get the bool env variable from env store
func (c *provider) GetBoolStoreEnvVariable(key string) (bool, error) {
	data, err := c.store.HGet(c.ctx, envStorePrefix, key).Result()
	if err != nil {
		return false, nil
	}

	return data == "1", nil
}
