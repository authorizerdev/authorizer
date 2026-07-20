package testwebhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Dependencies for the test webhook SMS provider.
type Dependencies struct {
	Log *zerolog.Logger
}

// testWebhookProvider is an sms.Provider that POSTs the plaintext SMS
// payload to a configured HTTP endpoint instead of calling a real carrier.
// Only ever wired when cfg.TestSMSWebhookURL is explicitly set — see
// internal/sms/provider.go. Exists purely for e2e-playground, where
// mocks/sms-sink stores the payload so Playwright tests can retrieve the
// OTP code that a real carrier would otherwise deliver by SMS.
type testWebhookProvider struct {
	webhookURL string
	client     *http.Client
	log        *zerolog.Logger
}

type payload struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// NewTestWebhookProvider constructs a test-only SMS provider.
func NewTestWebhookProvider(cfg *config.Config, deps *Dependencies) (*testWebhookProvider, error) {
	if cfg.TestSMSWebhookURL == "" {
		return nil, fmt.Errorf("TestSMSWebhookURL is required")
	}
	return &testWebhookProvider{
		webhookURL: cfg.TestSMSWebhookURL,
		client:     &http.Client{Timeout: 5 * time.Second},
		log:        deps.Log,
	}, nil
}

// SendSMS posts the plaintext code to the configured test webhook.
func (p *testWebhookProvider) SendSMS(sendTo, messageBody string) error {
	body, err := json.Marshal(payload{Phone: sendTo, Message: messageBody})
	if err != nil {
		return fmt.Errorf("failed to marshal test sms payload: %w", err)
	}
	resp, err := p.client.Post(p.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to post test sms webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("test sms webhook returned status %d", resp.StatusCode)
	}
	p.log.Debug().Str("phone", sendTo).Msg("test sms sent via webhook")
	return nil
}
