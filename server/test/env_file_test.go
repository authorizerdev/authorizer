package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/stretchr/testify/assert"
)

func TestEnvs(t *testing.T) {
	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyEnvPath, "../../.env.sample")
	env.InitAllEnv()
	store := envstore.EnvStoreObj.GetEnvStoreClone()

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
