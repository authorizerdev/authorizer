package sms

import (
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
)

// TestNew_TestWebhookURLUnset_NoTwilioCreds_ReturnsNilProvider proves the
// no-op claim for TestSMSWebhookURL: with it unset (production default) and
// no Twilio credentials configured either, New falls through to the exact
// same pre-branch outcome - no provider, no error - not some test-provider
// fallback.
func TestNew_TestWebhookURLUnset_NoTwilioCreds_ReturnsNilProvider(t *testing.T) {
	log := zerolog.Nop()
	provider, err := New(&config.Config{}, &Dependencies{Log: &log})
	require.NoError(t, err)
	assert.Nil(t, provider)
}

// TestNew_TestWebhookURLUnset_TwilioCredsSet_UsesTwilioProvider proves the
// Twilio branch is untouched when TestSMSWebhookURL is unset: real Twilio
// credentials still select the real Twilio provider, exactly as before this
// branch existed.
func TestNew_TestWebhookURLUnset_TwilioCredsSet_UsesTwilioProvider(t *testing.T) {
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

// TestNew_TestWebhookURLSet_UsesTestWebhookProvider confirms the test
// webhook takes priority over Twilio when explicitly configured, even if
// Twilio credentials also happen to be set.
func TestNew_TestWebhookURLSet_UsesTestWebhookProvider(t *testing.T) {
	log := zerolog.Nop()
	provider, err := New(&config.Config{
		TestSMSWebhookURL: "http://sms-sink.internal/sms",
		TwilioAPIKey:      "key",
		TwilioAPISecret:   "secret",
		TwilioAccountSID:  "sid",
		TwilioSender:      "+15550001111",
	}, &Dependencies{Log: &log})
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.NotContains(t, fmt.Sprintf("%T", provider), "twilio.")
}
