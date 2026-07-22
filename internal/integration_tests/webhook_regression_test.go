package integration_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSignupStillFiresWebhookEndToEnd is a regression check that
// internal/events.deliver's extraction (split out to share delivery mechanics
// with SCIM's provisioning-lifecycle events) didn't silently break the
// existing signup/login webhook path: register a real webhook for
// user.created, sign a user up, and confirm a log entry actually landed.
func TestSignupStillFiresWebhookEndToEnd(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	addedWebhook, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
		EventName:        constants.UserCreatedWebhookEvent,
		EventDescription: refs.NewStringRef("regression check"),
		Endpoint:         "https://example.invalid/webhook",
		Enabled:          true,
		Headers:          map[string]any{},
	})
	require.NoError(t, err)
	require.NotNil(t, addedWebhook)

	email := "webhook_regression_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes)

	// RegisterEvent fires via asyncutil.Go; drain it before asserting.
	asyncutil.Wait(zerolog.Nop())

	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
	logs, err := ts.GraphQLProvider.WebhookLogs(ctx, &model.ListWebhookLogRequest{
		Pagination: &model.PaginationRequest{Limit: refs.NewInt64Ref(50)},
	})
	require.NoError(t, err)
	require.NotNil(t, logs)
	require.NotEmpty(t, logs.WebhookLogs, "expected at least one webhook log after the signup that just happened")

	found := false
	for _, l := range logs.WebhookLogs {
		if l.Request != nil && strings.Contains(*l.Request, email) {
			found = true
			assert.Contains(t, *l.Request, `"event_name":"`+constants.UserCreatedWebhookEvent+`"`)
			break
		}
	}
	assert.True(t, found, "no webhook log request body referenced the just-signed-up user's email — signup no longer fires RegisterEvent as before")
}
