package inmemory

import "strings"

// ClearStore clears the in-memory store.
func (c *provider) ClearStore() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.sessionStore = map[string]map[string]string{}

	return nil
}

// GetUserSessions returns all the user session token from the in-memory store.
func (c *provider) GetUserSessions(userId string) map[string]string {
	// c.mutex.Lock()
	// defer c.mutex.Unlock()
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
	// c.mutex.Lock()
	// defer c.mutex.Unlock()
	sessions := c.GetUserSessions(userId)
	for k := range sessions {
		c.RemoveState(k)
	}

	return nil
}

// SetState sets the state in the in-memory store.
func (c *provider) SetState(key, state string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.stateStore[key] = state

	return nil
}

// GetState gets the state from the in-memory store.
func (c *provider) GetState(key string) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	state := ""
	if stateVal, ok := c.stateStore[key]; ok {
		state = stateVal
	}

	return state
}

// RemoveState removes the state from the in-memory store.
func (c *provider) RemoveState(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.stateStore, key)

	return nil
}
