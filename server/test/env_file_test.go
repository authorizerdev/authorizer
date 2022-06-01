package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

func TestEnvs(t *testing.T) {
	err := os.Setenv(constants.EnvKeyEnvPath, "../../.env.test")
	assert.Nil(t, err)
	err = memorystore.InitRequiredEnv()
	assert.Nil(t, err)
	err = env.InitAllEnv()
	assert.Nil(t, err)
	store, err := memorystore.Provider.GetEnvStore()
	assert.Nil(t, err)

	assert.Equal(t, "test", store[constants.EnvKeyEnv].(string))
	assert.False(t, store[constants.EnvKeyDisableEmailVerification].(bool))
	assert.False(t, store[constants.EnvKeyDisableMagicLinkLogin].(bool))
	assert.False(t, store[constants.EnvKeyDisableBasicAuthentication].(bool))
	assert.Equal(t, "RS256", store[constants.EnvKeyJwtType].(string))
	assert.Equal(t, store[constants.EnvKeyJwtRoleClaim].(string), "role")
	assert.EqualValues(t, store[constants.EnvKeyRoles].(string), "user")
	assert.EqualValues(t, store[constants.EnvKeyDefaultRoles].(string), "user")
	assert.EqualValues(t, store[constants.EnvKeyAllowedOrigins].(string), "*")
}
