package sms

import (
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/rs/zerolog"
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

// New returns a new sms provider
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	return nil, nil
}
