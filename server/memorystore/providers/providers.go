package providers

// Provider defines current memory store provider
type Provider interface {
	// DeleteAllSessions deletes all the sessions from the session store
	DeleteAllUserSession(userId string) error
	// GetUserSessions returns all the user sessions from the session store
	GetUserSessions(userId string) map[string]string
	// ClearStore clears the session store for authorizer tokens
	ClearStore() error
	// SetState sets the login state (key, value form) in the session store
	SetState(key, state string) error
	// GetState returns the state from the session store
	GetState(key string) (string, error)
	// RemoveState removes the social login state from the session store
	RemoveState(key string) error

	// methods for env store

	// UpdateEnvStore to update the whole env store object
	UpdateEnvStore(store map[string]interface{}) error
	// GetEnvStore() returns the env store object
	GetEnvStore() (map[string]interface{}, error)
	// UpdateEnvVariable to update the particular env variable
	UpdateEnvVariable(key string, value interface{}) error
	// GetStringStoreEnvVariable to get the string env variable from env store
	GetStringStoreEnvVariable(key string) (string, error)
	// GetBoolStoreEnvVariable to get the bool env variable from env store
	GetBoolStoreEnvVariable(key string) (bool, error)
	// GetSliceStoreEnvVariable to get the string slice env variable from env store
	GetSliceStoreEnvVariable(key string) ([]string, error)
}
