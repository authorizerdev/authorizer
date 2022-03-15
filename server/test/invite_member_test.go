package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func inviteUserTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should invite user successfully`, func(t *testing.T) {
		req, ctx := createContext(s)
		emails := []string{"invite_member1." + s.TestInfo.Email}

		// unauthorized error
		res, err := resolvers.InviteMembersResolver(ctx, model.InviteMemberInput{
			Emails: emails,
		})

		assert.Error(t, err)
		assert.Nil(t, res)

		h, err := crypto.EncryptPassword(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))

		// invalid emails test
		invalidEmailsTest := []string{
			"test",
			"test.com",
		}
		res, err = resolvers.InviteMembersResolver(ctx, model.InviteMemberInput{
			Emails: invalidEmailsTest,
		})

		// valid test
		res, err = resolvers.InviteMembersResolver(ctx, model.InviteMemberInput{
			Emails: emails,
		})
		assert.Nil(t, err)
		assert.NotNil(t, res)

		// duplicate error test
		res, err = resolvers.InviteMembersResolver(ctx, model.InviteMemberInput{
			Emails: emails,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		cleanData(emails[0])
	})
}
