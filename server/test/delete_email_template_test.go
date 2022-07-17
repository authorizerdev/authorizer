package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func deleteEmailTemplateTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should delete email templates", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		// get all email templates
		emailTemplates, err := db.Provider.ListEmailTemplate(ctx, model.Pagination{
			Limit:  10,
			Page:   1,
			Offset: 0,
		})
		assert.NoError(t, err)

		for _, e := range emailTemplates.EmailTemplates {
			res, err := resolvers.DeleteEmailTemplateResolver(ctx, model.DeleteEmailTemplateRequest{
				ID: e.ID,
			})

			assert.NoError(t, err)
			assert.NotNil(t, res)
			assert.NotEmpty(t, res.Message)
		}

		emailTemplates, err = db.Provider.ListEmailTemplate(ctx, model.Pagination{
			Limit:  10,
			Page:   1,
			Offset: 0,
		})
		assert.NoError(t, err)
		assert.Len(t, emailTemplates.EmailTemplates, 0)
	})
}
