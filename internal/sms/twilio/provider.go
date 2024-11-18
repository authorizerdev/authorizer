package twilio

import (
	"fmt"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/rs/zerolog"
)

// Dependencies struct for twilio provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       config.Config
	dependencies Dependencies
}

// NewTwilioProvider returns a new twilio provider
func NewTwilioProvider(cfg config.Config, deps Dependencies) (*provider, error) {
	if cfg.TwilioAPIKey == "" {
		deps.Log.Debug().Msg("missing twilio api key")
		return nil, fmt.Errorf("missing twilio api key")
	}
	if cfg.TwilioAPISecret == "" {
		deps.Log.Debug().Msg("missing twilio api secret")
		return nil, fmt.Errorf("missing twilio api secret")
	}
	if cfg.TwilioSender == "" {
		deps.Log.Debug().Msg("missing twilio sender")
		return nil, fmt.Errorf("missing twilio sender")
	}
	if cfg.TwilioAccountSID == "" {
		deps.Log.Debug().Msg("missing twilio account sid")
		return nil, fmt.Errorf("missing twilio account sid")
	}
	return &provider{
		config:       cfg,
		dependencies: deps,
	}, nil
}
