package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func emailTemplatesTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should get email templates", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		emailTemplates, err := resolvers.EmailTemplatesResolver(ctx, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, emailTemplates)
		assert.Len(t, emailTemplates.EmailTemplates, len(s.TestInfo.TestEmailTemplateEventTypes))
	})
}
