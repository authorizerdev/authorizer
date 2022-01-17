package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/stretchr/testify/assert"
)

func TestEnvs(t *testing.T) {
	envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.EnvKeyEnvPath, "../../.env.sample")
	env.InitEnv()
	store := envstore.EnvInMemoryStoreObj.GetEnvStoreClone()

	assert.Equal(t, store[constants.EnvKeyAdminSecret].(string), "admin")
	assert.Equal(t, store[constants.EnvKeyEnv].(string), "production")
	assert.False(t, store[constants.EnvKeyDisableEmailVerification].(bool))
	assert.False(t, store[constants.EnvKeyDisableMagicLinkLogin].(bool))
	assert.False(t, store[constants.EnvKeyDisableBasicAuthentication].(bool))
	assert.Equal(t, store[constants.EnvKeyJwtType].(string), "HS256")
	assert.Equal(t, store[constants.EnvKeyJwtSecret].(string), "random_string")
	assert.Equal(t, store[constants.EnvKeyJwtRoleClaim].(string), "role")
	assert.EqualValues(t, store[constants.EnvKeyRoles].([]string), []string{"user"})
	assert.EqualValues(t, store[constants.EnvKeyDefaultRoles].([]string), []string{"user"})
	assert.EqualValues(t, store[constants.EnvKeyProtectedRoles].([]string), []string{"admin"})
	assert.EqualValues(t, store[constants.EnvKeyAllowedOrigins].([]string), []string{"*"})
}
