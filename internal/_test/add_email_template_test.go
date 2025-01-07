package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func addEmailTemplateTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should add email templates", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		t.Run("should not add email template for invalid event type", func(t *testing.T) {
			emailTemplate, err := resolvers.AddEmailTemplateResolver(ctx, model.AddEmailTemplateRequest{
				EventName: "test",
			})
			assert.Error(t, err)
			assert.Nil(t, emailTemplate)
		})

		t.Run("should not add email template for empty subject", func(t *testing.T) {
			emailTemplate, err := resolvers.AddEmailTemplateResolver(ctx, model.AddEmailTemplateRequest{
				EventName: s.TestInfo.TestEmailTemplateEventTypes[0],
				Template:  " test ",
				Subject:   "    ",
			})
			assert.Error(t, err)
			assert.Nil(t, emailTemplate)
		})

		t.Run("should not add email template for empty template", func(t *testing.T) {
			emailTemplate, err := resolvers.AddEmailTemplateResolver(ctx, model.AddEmailTemplateRequest{
				EventName: s.TestInfo.TestEmailTemplateEventTypes[0],
				Template:  "     ",
				Subject:   "test",
			})
			assert.Error(t, err)
			assert.Nil(t, emailTemplate)
		})

		design := ""

		for _, eventType := range s.TestInfo.TestEmailTemplateEventTypes {
			t.Run("should add email template with empty design for "+eventType, func(t *testing.T) {
				emailTemplate, err := resolvers.AddEmailTemplateResolver(ctx, model.AddEmailTemplateRequest{
					EventName: eventType,
					Template:  "Test email",
					Subject:   "Test email",
					Design:    &design,
				})
				assert.NoError(t, err)
				assert.NotNil(t, emailTemplate)
				assert.NotEmpty(t, emailTemplate.Message)

				et, err := db.Provider.GetEmailTemplateByEventName(ctx, eventType)
				assert.NoError(t, err)
				assert.Equal(t, et.EventName, eventType)
				assert.Equal(t, "Test email", et.Subject)
				assert.Equal(t, "", et.Design)
			})
		}
	})
}
