package sms

import (
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/internalerr"
	"github.com/authorizerdev/authorizer/internal/sms/testwebhook"
	"github.com/authorizerdev/authorizer/internal/sms/twilio"
)

// Dependencies for sms provider
type Dependencies struct {
	Log *zerolog.Logger
}

// Provider interface to send sms
type Provider interface {
	// SendSMS sends sms
	SendSMS(sendTo, messageBody string) error
}

// errorBoundary marks every SMS-provider error internal (internalerr.Wrap):
// the text of a Twilio/webhook failure can carry account and API details and
// must never reach an API client verbatim.
type errorBoundary struct {
	next Provider
}

// SendSMS sends sms through the wrapped provider.
func (p *errorBoundary) SendSMS(sendTo, messageBody string) error {
	return internalerr.Wrap("sms", p.next.SendSMS(sendTo, messageBody))
}

// New returns a new sms provider. A nil provider (SMS not configured) stays
// nil so callers' "is SMS enabled" checks keep working.
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	next, err := newBackend(cfg, deps)
	if err != nil || next == nil {
		return next, err
	}
	return &errorBoundary{next: next}, nil
}

func newBackend(cfg *config.Config, deps *Dependencies) (Provider, error) {
	var provider Provider
	var err error
	if cfg.Env == constants.E2EEnv {
		return testwebhook.NewTestWebhookProvider(&testwebhook.Dependencies{Log: deps.Log})
	}
	if cfg.TwilioAPIKey != "" && cfg.TwilioAPISecret != "" && cfg.TwilioAccountSID != "" && cfg.TwilioSender != "" {
		provider, err = twilio.NewTwilioProvider(cfg, &twilio.Dependencies{
			Log: deps.Log,
		})
		if err != nil {
			return nil, err
		}
	}
	return provider, nil
}
