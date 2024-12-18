package token

import (
	"fmt"

	"github.com/golang-jwt/jwt"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Dependencies struct for twilio provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       *config.Config
	dependencies Dependencies
}

// NewTokenProvider returns a new token provider
func NewTokenProvider(cfg *config.Config, deps Dependencies) (*provider, error) {
	if cfg.JWTType == "" {
		deps.Log.Debug().Msg("missing jwt type")
		return nil, fmt.Errorf("missing jwt type")
	}
	signingMethod := jwt.GetSigningMethod(cfg.JWTType)
	switch signingMethod {
	case jwt.SigningMethodHS256, jwt.SigningMethodHS384, jwt.SigningMethodHS512:
		if cfg.JWTSecret == "" {
			deps.Log.Debug().Msg("missing jwt secret")
			return nil, fmt.Errorf("missing jwt secret")
		}
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512,
		jwt.SigningMethodES256, jwt.SigningMethodES384, jwt.SigningMethodES512:
		if cfg.JWTPrivateKey == "" {
			deps.Log.Debug().Msg("missing jwt private key")
			return nil, fmt.Errorf("missing jwt private key")
		}
		if cfg.JWTPublicKey == "" {
			deps.Log.Debug().Msg("missing jwt public key")
			return nil, fmt.Errorf("missing jwt public key")
		}
	}
	return &provider{
		config:       cfg,
		dependencies: deps,
	}, nil
}
