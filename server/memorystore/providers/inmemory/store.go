package inmemory

import (
	"fmt"
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
)

// ClearStore clears the in-memory store.
func (c *provider) ClearStore() error {
	if os.Getenv("ENV") != constants.TestEnv {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
	c.sessionStore = map[string]map[string]string{}

	return nil
}

// GetUserSessions returns all the user session token from the in-memory store.
func (c *provider) GetUserSessions(userId string) map[string]string {
	res := map[string]string{}
	for k, v := range c.stateStore {
		split := strings.Split(v, "@")
		if split[1] == userId {
			res[k] = split[0]
		}
	}

	return res
}

// DeleteAllUserSession deletes all the user sessions from in-memory store.
func (c *provider) DeleteAllUserSession(userId string) error {
	if os.Getenv("ENV") != constants.TestEnv {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
	sessions := c.GetUserSessions(userId)
	for k := range sessions {
		c.RemoveState(k)
	}

	return nil
}

// SetState sets the state in the in-memory store.
func (c *provider) SetState(key, state string) error {
	if os.Getenv("ENV") != constants.TestEnv {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
	c.stateStore[key] = state

	return nil
}

// GetState gets the state from the in-memory store.
func (c *provider) GetState(key string) (string, error) {
	state := ""
	if stateVal, ok := c.stateStore[key]; ok {
		state = stateVal
	}

	return state, nil
}

// RemoveState removes the state from the in-memory store.
func (c *provider) RemoveState(key string) error {
	if os.Getenv("ENV") != constants.TestEnv {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
	delete(c.stateStore, key)

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
