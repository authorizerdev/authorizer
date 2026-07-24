package testwebhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// e2eSMSSinkURL is the fixed docker-compose-internal address of the
// e2e-playground SMS sink mock (e2e-playground/mocks/sms-sink). Only ever
// reachable from inside that specific docker-compose network - never
// resolvable in a real deployment, and this provider is only ever
// constructed at all when Config.Env == constants.E2EEnv (see
// internal/sms/provider.go).
const e2eSMSSinkURL = "http://sms-sink:4100/sms"

// Dependencies for the test webhook SMS provider.
type Dependencies struct {
	Log *zerolog.Logger
}

// testWebhookProvider is an sms.Provider that POSTs the plaintext SMS
// payload to e2eSMSSinkURL instead of calling a real carrier. Only ever
// wired when Config.Env == constants.E2EEnv — see internal/sms/provider.go.
// Exists purely for e2e-playground, where mocks/sms-sink stores the payload
// so tests can retrieve the OTP code a real carrier would otherwise deliver.
type testWebhookProvider struct {
	webhookURL string
	client     *http.Client
	log        *zerolog.Logger
}

type payload struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// NewTestWebhookProvider constructs a test-only SMS provider. Callers must
// only construct this when Config.Env == constants.E2EEnv.
func NewTestWebhookProvider(deps *Dependencies) (*testWebhookProvider, error) {
	return &testWebhookProvider{
		webhookURL: e2eSMSSinkURL,
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
