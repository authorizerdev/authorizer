package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func updateEmailTemplateTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should update email template", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		// get email template
		emailTemplate, err := db.Provider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, emailTemplate)

		res, err := resolvers.UpdateEmailTemplateResolver(ctx, model.UpdateEmailTemplateRequest{
			ID:       emailTemplate.ID,
			Template: refs.NewStringRef("Updated test template"),
			Subject:  refs.NewStringRef("Updated subject"),
			Design:   refs.NewStringRef("Updated design"),
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, res)
		assert.NotEmpty(t, res.Message)

		updatedEmailTemplate, err := db.Provider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, updatedEmailTemplate)
		assert.Equal(t, emailTemplate.ID, updatedEmailTemplate.ID)
		assert.Equal(t, updatedEmailTemplate.Template, "Updated test template")
		assert.Equal(t, updatedEmailTemplate.Subject, "Updated subject")
		assert.Equal(t, updatedEmailTemplate.Design, "Updated design")
	})
}
