package redis

import (
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
)

var (
	// state store prefix
	stateStorePrefix = "authorizer_state:"
)

const mfaSessionPrefix = "mfa_session_"

// SetUserSession sets the user session for given user identifier in form recipe:user_id
func (p *provider) SetUserSession(userId, key, token string, expiration int64) error {
	currentTime := time.Now()
	expireTime := time.Unix(expiration, 0)
	duration := expireTime.Sub(currentTime)
	err := p.store.Set(p.ctx, fmt.Sprintf("%s:%s", userId, key), token, duration).Err()
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error saving user session to redis")
		return err
	}
	return nil
}

// GetUserSession returns the user session from redis store.
func (p *provider) GetUserSession(userId, key string) (string, error) {
	data, err := p.store.Get(p.ctx, fmt.Sprintf("%s:%s", userId, key)).Result()
	if err != nil {
		return "", err
	}
	return data, nil
}

// DeleteUserSession deletes the user session from redis store.
func (p *provider) DeleteUserSession(userId, key string) error {
	keys := []string{
		constants.TokenTypeSessionToken + "_" + key,
		constants.TokenTypeAccessToken + "_" + key,
		constants.TokenTypeRefreshToken + "_" + key,
	}
	for _, k := range keys {
		if err := p.store.Del(p.ctx, fmt.Sprintf("%s:%s", userId, k)).Err(); err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("Error deleting user session from redis")
			// continue
		}
	}

	return nil
}

// DeleteAllUserSessions deletes all the user session from redis
func (p *provider) DeleteAllUserSessions(userID string) error {
	res := p.store.Keys(p.ctx, fmt.Sprintf("*%s*", userID))
	if res.Err() != nil {
		p.dependencies.Log.Debug().Err(res.Err()).Msg("Error getting all user sessions from redis")
		return res.Err()
	}
	keys := res.Val()
	for _, key := range keys {
		err := p.store.Del(p.ctx, key).Err()
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("Error deleting all user sessions from redis")
			continue
		}
	}
	return nil
}

// DeleteSessionForNamespace to delete session for a given namespace example google,github
func (p *provider) DeleteSessionForNamespace(namespace string) error {
	res := p.store.Keys(p.ctx, fmt.Sprintf("%s:*", namespace))
	if res.Err() != nil {
		p.dependencies.Log.Debug().Err(res.Err()).Msg("Error getting all user sessions from redis")
		return res.Err()
	}
	keys := res.Val()
	for _, key := range keys {
		err := p.store.Del(p.ctx, key).Err()
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("Error deleting all user sessions from redis")
			continue
		}
	}
	return nil
}

// SetMfaSession sets the mfa session with key and value of userId
func (p *provider) SetMfaSession(userId, key string, expiration int64) error {
	currentTime := time.Now()
	expireTime := time.Unix(expiration, 0)
	duration := expireTime.Sub(currentTime)
	err := p.store.Set(p.ctx, fmt.Sprintf("%s%s:%s", mfaSessionPrefix, userId, key), userId, duration).Err()
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error saving mfa session to redis")
		return err
	}
	return nil
}

// GetMfaSession returns value of given mfa session
func (p *provider) GetMfaSession(userId, key string) (string, error) {
	data, err := p.store.Get(p.ctx, fmt.Sprintf("%s%s:%s", mfaSessionPrefix, userId, key)).Result()
	if err != nil {
		return "", err
	}
	return data, nil
}

// GetAllMfaSessions returns all mfa sessions for given userId
func (p *provider) GetAllMfaSessions(userId string) ([]string, error) {
	res := p.store.Keys(p.ctx, fmt.Sprintf("%s%s:*", mfaSessionPrefix, userId))
	if res.Err() != nil {
		p.dependencies.Log.Debug().Err(res.Err()).Msg("Error getting all mfa sessions from redis")
		return nil, res.Err()
	}
	keys := res.Val()
	for i := 0; i < len(keys); i++ {
		keys[i] = keys[i][len(mfaSessionPrefix)+len(userId)+1:]
	}
	return keys, nil
}

// DeleteMfaSession deletes given mfa session from in-memory store.
func (p *provider) DeleteMfaSession(userId, key string) error {
	if err := p.store.Del(p.ctx, fmt.Sprintf("%s%s:%s", mfaSessionPrefix, userId, key)).Err(); err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error deleting mfa session from redis")
		// continue
	}
	return nil
}

// SetState sets the state in redis store.
func (p *provider) SetState(key, value string) error {
	err := p.store.Set(p.ctx, stateStorePrefix+key, value, 0).Err()
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error saving state to redis")
		return err
	}

	return nil
}

// GetState gets the state from redis store.
func (p *provider) GetState(key string) (string, error) {
	data, err := p.store.Get(p.ctx, stateStorePrefix+key).Result()
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error getting state from redis")
		return "", err
	}

	return data, err
}

// RemoveState removes the state from redis store.
func (p *provider) RemoveState(key string) error {
	err := p.store.Del(p.ctx, stateStorePrefix+key).Err()
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error deleting state from redis")
		return err
	}

	return nil
}

// GetAllData returns all the data from the session store
// This is used for testing purposes only
func (p *provider) GetAllData() (map[string]string, error) {
	res := p.store.Keys(p.ctx, "*")
	if res.Err() != nil {
		p.dependencies.Log.Debug().Err(res.Err()).Msg("Error getting all data from redis")
		return nil, res.Err()
	}
	keys := res.Val()
	data := make(map[string]string)
	for _, key := range keys {
		val, err := p.store.Get(p.ctx, key).Result()
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("Error getting all data from redis")
			continue
		}
		data[key] = val
	}
	return data, nil
}
