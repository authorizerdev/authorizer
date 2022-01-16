package envstore

import (
	"sync"

	"github.com/authorizerdev/authorizer/server/constants"
)

// EnvInMemoryStore struct
type EnvInMemoryStore struct {
	mutex sync.Mutex
	store map[string]interface{}
}

// EnvInMemoryStoreObj global variable for EnvInMemoryStore
var EnvInMemoryStoreObj = &EnvInMemoryStore{
	store: map[string]interface{}{
		constants.EnvKeyAdminCookieName:            "authorizer-admin",
		constants.EnvKeyJwtRoleClaim:               "role",
		constants.EnvKeyOrganizationName:           "Authorizer",
		constants.EnvKeyOrganizationLogo:           "https://www.authorizer.io/images/logo.png",
		constants.EnvKeyDisableBasicAuthentication: false,
		constants.EnvKeyDisableMagicLinkLogin:      false,
		constants.EnvKeyDisableEmailVerification:   false,
		constants.EnvKeyDisableLoginPage:           false,
	},
}

// UpdateEnvStore to update the whole env store object
func (e *EnvInMemoryStore) UpdateEnvStore(data map[string]interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	// just override the keys + new keys
	for key, value := range data {
		e.store[key] = value
	}
}

// UpdateEnvVariable to update the particular env variable
func (e *EnvInMemoryStore) UpdateEnvVariable(key string, value interface{}) map[string]interface{} {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.store[key] = value
	return e.store
}

// GetEnvStore to get the env variable from env store object
func (e *EnvInMemoryStore) GetEnvVariable(key string) interface{} {
	// e.mutex.Lock()
	// defer e.mutex.Unlock()
	return e.store[key]
}

// GetEnvStoreClone to get clone of current env store object
func (e *EnvInMemoryStore) GetEnvStoreClone() map[string]interface{} {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	result := make(map[string]interface{})
	for key, value := range e.store {
		result[key] = value
	}

	return result
}
