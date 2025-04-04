package in_memory

import (
	"fmt"
	"os"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// SetUserSession sets the user session for given user identifier in form recipe:user_id
func (c *provider) SetUserSession(userId, key, token string, expiration int64) error {
	c.sessionStore.Set(userId, key, token, expiration)
	return nil
}

// GetUserSession returns value for given session token
func (c *provider) GetUserSession(userId, sessionToken string) (string, error) {
	val := c.sessionStore.Get(userId, sessionToken)
	if val == "" {
		return "", fmt.Errorf("Not found")
	}
	return val, nil
}

// DeleteAllUserSessions deletes all the user sessions from in-memory store.
func (c *provider) DeleteAllUserSessions(userId string) error {
	c.sessionStore.RemoveAll(userId)
	return nil
}

// DeleteUserSession deletes the user session from the in-memory store.
func (c *provider) DeleteUserSession(userId, sessionToken string) error {
	c.sessionStore.Remove(userId, constants.TokenTypeSessionToken+"_"+sessionToken)
	c.sessionStore.Remove(userId, constants.TokenTypeAccessToken+"_"+sessionToken)
	c.sessionStore.Remove(userId, constants.TokenTypeRefreshToken+"_"+sessionToken)
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
		return "", fmt.Errorf("Not found")
	}
	return val, nil
}

// GetAllMfaSessions returns all mfa sessions for given userId
func (p *provider) GetAllMfaSessions(userId string) ([]string, error) {
	values := p.mfasessionStore.GetAll(userId)
	if len(values) == 0 {
		return nil, fmt.Errorf("Not found")
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
	if os.Getenv("ENV") != constants.TestEnv {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
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
