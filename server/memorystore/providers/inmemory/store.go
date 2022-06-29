package inmemory

import (
	"fmt"
	"os"

	"github.com/authorizerdev/authorizer/server/constants"
)

// SetUserSession sets the user session
func (c *provider) SetUserSession(userId, key, token string) error {
	c.sessionStore.Set(userId, key, token)
	return nil
}

// GetAllUserSessions returns all the user sessions token from the in-memory store.
func (c *provider) GetAllUserSessions(userId string) (map[string]string, error) {
	data := c.sessionStore.GetAll(userId)
	return data, nil
}

// GetUserSession returns value for given session token
func (c *provider) GetUserSession(userId, sessionToken string) (string, error) {
	return c.sessionStore.Get(userId, sessionToken), nil
}

// DeleteAllUserSessions deletes all the user sessions from in-memory store.
func (c *provider) DeleteAllUserSessions(userId string) error {
	namespaces := []string{
		constants.AuthRecipeMethodBasicAuth,
		constants.AuthRecipeMethodMagicLinkLogin,
		constants.AuthRecipeMethodApple,
		constants.AuthRecipeMethodFacebook,
		constants.AuthRecipeMethodGithub,
		constants.AuthRecipeMethodGoogle,
		constants.AuthRecipeMethodLinkedIn,
	}
	if os.Getenv("ENV") != constants.TestEnv {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
	for _, namespace := range namespaces {
		c.sessionStore.RemoveAll(namespace + ":" + userId)
	}
	return nil
}

// DeleteUserSession deletes the user session from the in-memory store.
func (c *provider) DeleteUserSession(userId, sessionToken string) error {
	if os.Getenv("ENV") != constants.TestEnv {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
	c.sessionStore.Remove(userId, sessionToken)
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

// UpdateEnvStore to update the whole env store object
func (c *provider) UpdateEnvStore(store map[string]interface{}) error {
	c.envStore.UpdateStore(store)
	return nil
}

// GetEnvStore returns the env store object
func (c *provider) GetEnvStore() (map[string]interface{}, error) {
	return c.envStore.GetStore(), nil
}

// UpdateEnvVariable to update the particular env variable
func (c *provider) UpdateEnvVariable(key string, value interface{}) error {
	c.envStore.Set(key, value)
	return nil
}

// GetStringStoreEnvVariable to get the env variable from string store object
func (c *provider) GetStringStoreEnvVariable(key string) (string, error) {
	res := c.envStore.Get(key)
	if res == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", res), nil
}

// GetBoolStoreEnvVariable to get the env variable from bool store object
func (c *provider) GetBoolStoreEnvVariable(key string) (bool, error) {
	res := c.envStore.Get(key)
	if res == nil {
		return false, nil
	}
	return res.(bool), nil
}
