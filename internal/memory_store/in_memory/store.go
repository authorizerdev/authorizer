package in_memory

import (
	"fmt"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// SetUserSession sets the user session for given user identifier in form recipe:user_id
func (c *provider) SetUserSession(userId, key, token string, expiration int64) error {
	c.sessionStore.Set(userId, key, token, expiration)
	return nil
}

// GetUserSession returns value for given session token
func (c *provider) GetUserSession(userId, key string) (string, error) {
	val := c.sessionStore.Get(userId, key)
	if val == "" {
		return "", fmt.Errorf("not found")
	}
	return val, nil
}

// DeleteAllUserSessions deletes all the user sessions from in-memory store.
func (c *provider) DeleteAllUserSessions(userId string) error {
	c.sessionStore.RemoveAll(userId)
	return nil
}

// DeleteUserSession deletes the user session from the in-memory store.
func (c *provider) DeleteUserSession(userId, key string) error {
	keys := []string{
		constants.TokenTypeSessionToken + "_" + key,
		constants.TokenTypeAccessToken + "_" + key,
		constants.TokenTypeRefreshToken + "_" + key,
	}

	for _, k := range keys {
		c.sessionStore.Remove(userId, k)
	}
	return nil
}

// DeleteSessionForNamespace to delete session for a given namespace example google,github
func (c *provider) DeleteSessionForNamespace(namespace string) error {
	c.sessionStore.RemoveByNamespace(namespace)
	return nil
}

// SetMfaSession sets the mfa session with key and value of userId
func (c *provider) SetMfaSession(userId, key string, expiration int64) error {
	c.mfasessionStore.Set(userId, key, userId, expiration)
	return nil
}

// GetMfaSession returns value of given mfa session
func (c *provider) GetMfaSession(userId, key string) (string, error) {
	val := c.mfasessionStore.Get(userId, key)
	if val == "" {
		return "", fmt.Errorf("not found")
	}
	return val, nil
}

// GetAllMfaSessions returns all mfa sessions for given userId
func (p *provider) GetAllMfaSessions(userId string) ([]string, error) {
	values := p.mfasessionStore.GetAll(userId)
	if len(values) == 0 {
		return nil, fmt.Errorf("not found")
	}
	return values, nil
}

// DeleteMfaSession deletes given mfa session from in-memory store.
func (c *provider) DeleteMfaSession(userId, key string) error {
	c.mfasessionStore.Remove(userId, key)
	return nil
}

// SetState sets the state in the in-memory store.
func (c *provider) SetState(key, state string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.stateStore.Set(key, state)

	return nil
}

// GetState gets the state from the in-memory store.
func (c *provider) GetState(key string) (string, error) {
	return c.stateStore.Get(key), nil
}

// RemoveState removes the state from the in-memory store.
func (c *provider) RemoveState(key string) error {
	c.stateStore.Remove(key)
	return nil
}

// GetAllData returns all the data from the in-memory store
// This is used for testing purposes only
func (c *provider) GetAllData() (map[string]string, error) {
	// Get all data from the session store and mfa session store
	// and merge them into a single map
	data := make(map[string]string)
	for k, v := range c.sessionStore.GetAllData() {
		data[k] = v
	}
	for k, v := range c.mfasessionStore.GetAllData() {
		data[k] = v
	}
	return data, nil
}
