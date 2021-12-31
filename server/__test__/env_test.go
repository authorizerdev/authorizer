package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/stretchr/testify/assert"
)

func TestEnvs(t *testing.T) {
	constants.EnvData.ENV_PATH = "../../.env.sample"

	assert.Equal(t, constants.EnvData.ADMIN_SECRET, "admin")
	assert.Equal(t, constants.EnvData.ENV, "production")
	assert.False(t, constants.EnvData.DISABLE_EMAIL_VERIFICATION)
	assert.False(t, constants.EnvData.DISABLE_MAGIC_LINK_LOGIN)
	assert.False(t, constants.EnvData.DISABLE_BASIC_AUTHENTICATION)
	assert.Equal(t, constants.EnvData.JWT_TYPE, "HS256")
	assert.Equal(t, constants.EnvData.JWT_SECRET, "random_string")
	assert.Equal(t, constants.EnvData.JWT_ROLE_CLAIM, "role")
	assert.EqualValues(t, constants.EnvData.ROLES, []string{"user"})
	assert.EqualValues(t, constants.EnvData.DEFAULT_ROLES, []string{"user"})
	assert.EqualValues(t, constants.EnvData.PROTECTED_ROLES, []string{"admin"})
	assert.EqualValues(t, constants.EnvData.ALLOWED_ORIGINS, []string{"*"})
}
