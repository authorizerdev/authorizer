package token

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/memory_store"
)

// Dependencies struct for twilio provider
type Dependencies struct {
	Log                 *zerolog.Logger
	MemoryStoreProvider memory_store.Provider
}

type provider struct {
	config       *config.Config
	dependencies *Dependencies
}

var _ Provider = &provider{}

// Provider interface for token provider
type Provider interface {
	// CreateAccessToken creates an access token
	CreateAccessToken(cfg *AuthTokenConfig) (string, int64, error)
	// CreateAuthToken creates all types of auth token
	CreateAuthToken(gc *gin.Context, cfg *AuthTokenConfig) (*AuthToken, error)
	// CreateIDToken creates an id token
	CreateIDToken(cfg *AuthTokenConfig) (string, int64, error)
	// CreateRefreshToken creates a refresh token
	CreateRefreshToken(cfg *AuthTokenConfig) (string, int64, error)
	// CreateSessionToken creates a session token
	CreateSessionToken(cfg *AuthTokenConfig) (*SessionData, string, int64, error)
	// CreateVerificationToken creates a verification token
	CreateVerificationToken(authTokenConfig *AuthTokenConfig, redirectURL string, tokenType string) (string, error)
	// GetAd
	GetAdminAuthToken(gc *gin.Context) (string, error)
	// GetAccessToken gets access token from request
	GetAccessToken(gc *gin.Context) (string, error)
	// GetIDToken gets id token from request
	GetIDToken(gc *gin.Context) (string, error)
	// GetUserIDFromSessionOrAccessToken gets user id from session or access token
	GetUserIDFromSessionOrAccessToken(gc *gin.Context) (*SessionOrAccessTokenData, error)
	// IsSuperAdmin checks if user is super admin
	IsSuperAdmin(gc *gin.Context) bool
	// ParseJWTToken parses jwt token
	ParseJWTToken(token string) (jwt.MapClaims, error)
	// SignJWTToken signs jwt token
	SignJWTToken(jwtclaims jwt.MapClaims) (string, error)
	// ValidateAccessToken validates access token
	ValidateAccessToken(gc *gin.Context, accessToken string) (map[string]interface{}, error)
	// ValidateAdminToken validates session token
	ValidateBrowserSession(gc *gin.Context, encryptedSession string) (*SessionData, error)
	// ValidateJWTClaims validates jwt claims
	ValidateJWTClaims(claims jwt.MapClaims, authTokenConfig *AuthTokenConfig) (bool, error)
	// ValidateJWTTokenWithoutNonce validates jwt token without nonce
	ValidateJWTTokenWithoutNonce(claims jwt.MapClaims, authTokenConfig *AuthTokenConfig) (bool, error)
	// ValidateRefreshToken validates refresh token
	ValidateRefreshToken(gc *gin.Context, refreshToken string) (map[string]interface{}, error)
}

// New returns a new token provider
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
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
