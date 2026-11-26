package email

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/internalerr"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// noTemplateStorage makes the provider fall back to the built-in template. The
// embedded nil Provider satisfies the rest of the interface; SendEmail must
// not reach it.
type noTemplateStorage struct{ storage.Provider }

func (noTemplateStorage) GetEmailTemplateByEventName(context.Context, string) (*schemas.EmailTemplate, error) {
	return nil, errors.New("not found")
}

// TestSendEmail_SMTPErrorIsInternal: an SMTP failure carries the SMTP host and
// server responses, so it is marked internal and masked by the transports
// (issue #356).
func TestSendEmail_SMTPErrorIsInternal(t *testing.T) {
	log := zerolog.Nop()
	p, err := New(&config.Config{
		SMTPHost:        "127.0.0.1",
		SMTPPort:        1, // nothing listens here: dial fails immediately
		SMTPSenderEmail: "noreply@example.com",
	}, &Dependencies{Log: &log, StorageProvider: noTemplateStorage{}})
	require.NoError(t, err)

	err = p.SendEmail([]string{"user@example.com"}, constants.VerificationTypeOTP, map[string]interface{}{
		"user":         map[string]interface{}{},
		"organization": map[string]interface{}{},
		"otp":          "123456",
	})
	require.Error(t, err)
	assert.True(t, internalerr.Is(err))
	assert.Equal(t, "email", internalerr.Source(err))
}
