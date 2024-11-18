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

const mfaSessionPrefix = "mfa_sess_"

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
	if err := p.store.Del(p.ctx, fmt.Sprintf("%s:%s", userId, constants.TokenTypeSessionToken+"_"+key)).Err(); err != nil {

		// continue
	}
	if err := p.store.Del(p.ctx, fmt.Sprintf("%s:%s", userId, constants.TokenTypeAccessToken+"_"+key)).Err(); err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error deleting user session from redis")
		// continue
	}
	if err := p.store.Del(p.ctx, fmt.Sprintf("%s:%s", userId, constants.TokenTypeRefreshToken+"_"+key)).Err(); err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error deleting user session from redis")
		// continue
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
