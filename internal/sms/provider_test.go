package sms

import (
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestNew_ProductionDefault_NoTwilioCreds_ReturnsNilProvider proves the no-op
// claim: with Env unset (production default) and no Twilio credentials
// configured either, New falls through to the exact same pre-branch outcome
// - no provider, no error - not some test-provider fallback.
func TestNew_ProductionDefault_NoTwilioCreds_ReturnsNilProvider(t *testing.T) {
	log := zerolog.Nop()
	provider, err := New(&config.Config{}, &Dependencies{Log: &log})
	require.NoError(t, err)
	assert.Nil(t, provider)
}

// TestNew_ProductionDefault_TwilioCredsSet_UsesTwilioProvider proves the
// Twilio branch is untouched when Env is unset: real Twilio credentials
// still select the real Twilio provider, exactly as before this branch
// existed.
func TestNew_ProductionDefault_TwilioCredsSet_UsesTwilioProvider(t *testing.T) {
	log := zerolog.Nop()
	provider, err := New(&config.Config{
		TwilioAPIKey:     "key",
		TwilioAPISecret:  "secret",
		TwilioAccountSID: "sid",
		TwilioSender:     "+15550001111",
	}, &Dependencies{Log: &log})
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Contains(t, fmt.Sprintf("%T", provider), "twilio.")
}

// TestNew_TestEnv_StillUsesTwilioProvider proves E2EEnv and TestEnv are
// genuinely distinct: internal/integration_tests runs with Env=TestEnv and
// must not accidentally start routing SMS through the e2e-playground sink.
func TestNew_TestEnv_StillUsesTwilioProvider(t *testing.T) {
	log := zerolog.Nop()
	provider, err := New(&config.Config{
		Env:              constants.TestEnv,
		TwilioAPIKey:     "key",
		TwilioAPISecret:  "secret",
		TwilioAccountSID: "sid",
		TwilioSender:     "+15550001111",
	}, &Dependencies{Log: &log})
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Contains(t, fmt.Sprintf("%T", provider), "twilio.")
}

// TestNew_E2EEnv_UsesTestWebhookProvider confirms the escape hatch: under
// --env=e2e, the test webhook provider is selected - even if Twilio
// credentials also happen to be set.
func TestNew_E2EEnv_UsesTestWebhookProvider(t *testing.T) {
	log := zerolog.Nop()
	provider, err := New(&config.Config{
		Env:              constants.E2EEnv,
		TwilioAPIKey:     "key",
		TwilioAPISecret:  "secret",
		TwilioAccountSID: "sid",
		TwilioSender:     "+15550001111",
	}, &Dependencies{Log: &log})
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.NotContains(t, fmt.Sprintf("%T", provider), "twilio.")
}
