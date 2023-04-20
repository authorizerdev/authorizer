package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
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

		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)

		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		// invalid emails test
		invalidEmailsTest := []string{
			"test",
			"test.com",
		}
		res, err = resolvers.InviteMembersResolver(ctx, model.InviteMemberInput{
			Emails: invalidEmailsTest,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		// valid test
		res, err = resolvers.InviteMembersResolver(ctx, model.InviteMemberInput{
			Emails: emails,
		})
		assert.Nil(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res.Message)
		assert.NotNil(t, res.Users)
		// duplicate error test
		res, err = resolvers.InviteMembersResolver(ctx, model.InviteMemberInput{
			Emails: emails,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		cleanData(emails[0])
	})
}
