package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func TestEnvs(t *testing.T) {
	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyEnvPath, "../../.env.sample")
	env.InitAllEnv()
	store, err := memorystore.Provider.GetEnvStore()
	assert.Nil(t, err)

	assert.Equal(t, store[constants.EnvKeyEnv].(string), "production")
	assert.False(t, store[constants.EnvKeyDisableEmailVerification].(bool))
	assert.False(t, store[constants.EnvKeyDisableMagicLinkLogin].(bool))
	assert.False(t, store[constants.EnvKeyDisableBasicAuthentication].(bool))
	assert.Equal(t, store[constants.EnvKeyJwtType].(string), "RS256")
	assert.Equal(t, store[constants.EnvKeyJwtRoleClaim].(string), "role")
	assert.EqualValues(t, utils.ConvertInterfaceToStringSlice(store[constants.EnvKeyRoles]), []string{"user"})
	assert.EqualValues(t, utils.ConvertInterfaceToStringSlice(store[constants.EnvKeyDefaultRoles]), []string{"user"})
	assert.EqualValues(t, utils.ConvertInterfaceToStringSlice(store[constants.EnvKeyAllowedOrigins]), []string{"*"})
}
