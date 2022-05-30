package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/stretchr/testify/assert"
)

func TestEnvs(t *testing.T) {
	memorystore.Provider.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyEnvPath, "../../.env.sample")
	env.InitAllEnv()
	store := memorystore.Provider.GetEnvStoreClone()

	assert.Equal(t, store.StringEnv[constants.EnvKeyEnv], "production")
	assert.False(t, store.BoolEnv[constants.EnvKeyDisableEmailVerification])
	assert.False(t, store.BoolEnv[constants.EnvKeyDisableMagicLinkLogin])
	assert.False(t, store.BoolEnv[constants.EnvKeyDisableBasicAuthentication])
	assert.Equal(t, store.StringEnv[constants.EnvKeyJwtType], "RS256")
	assert.Equal(t, store.StringEnv[constants.EnvKeyJwtRoleClaim], "role")
	assert.EqualValues(t, store.SliceEnv[constants.EnvKeyRoles], []string{"user"})
	assert.EqualValues(t, store.SliceEnv[constants.EnvKeyDefaultRoles], []string{"user"})
	assert.EqualValues(t, store.SliceEnv[constants.EnvKeyAllowedOrigins], []string{"*"})
}
