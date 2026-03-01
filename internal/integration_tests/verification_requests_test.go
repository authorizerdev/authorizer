package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerificationRequests tests the _verification_requests admin query
func TestVerificationRequests(t *testing.T) {
	cfg := getTestConfig()
	cfg.SMTPHost = "localhost"
	cfg.SMTPPort = 1025
	cfg.SMTPSenderEmail = "test@authorizer.dev"
	cfg.SMTPSenderName = "Test"
	cfg.SMTPLocalName = "Test"
	cfg.SMTPSkipTLSVerification = true
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	t.Run("should fail without admin auth", func(t *testing.T) {
		req.Header.Set("Cookie", "")
		res, err := ts.GraphQLProvider.VerificationRequests(ctx, &model.PaginatedRequest{})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should list verification requests with admin auth", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		res, err := ts.GraphQLProvider.VerificationRequests(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res.Pagination)
	})
}
