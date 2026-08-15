package token

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
)

// Dependencies struct for token provider
type Dependencies struct {
	Log                 *zerolog.Logger
	MemoryStoreProvider memory_store.Provider
	// StorageProvider backs the revocation re-check in ValidateAccessToken /
	// ValidateBrowserSession: defense-in-depth so a deprovisioned user
	// (RevokedTimestamp set — SCIM active:false, account deactivation) loses
	// request-serving access even if the session-store delete that normally
	// invalidates the token was missed or failed on this instance.
	StorageProvider storage.Provider
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
	// CreateMachineAuthToken creates a client_credentials access token (RFC 6749 §4.4)
	CreateMachineAuthToken(cfg *AuthTokenConfig) (*JWTToken, error)
	// CreateDelegatedAccessToken creates an RFC 8693 delegation access token
	// (nested `act` chain, resource-bound `aud`, short TTL).
	CreateDelegatedAccessToken(cfg *DelegationTokenConfig) (*JWTToken, error)
	// CreateIDToken creates an id token
	CreateIDToken(cfg *AuthTokenConfig) (string, int64, error)
	// CreateRefreshToken creates a refresh token
	CreateRefreshToken(cfg *AuthTokenConfig) (string, int64, error)
	// CreateSessionToken creates a session token
	CreateSessionToken(cfg *AuthTokenConfig) (*SessionData, string, int64, error)
	// CreateVerificationToken creates a verification token
	CreateVerificationToken(authTokenConfig *AuthTokenConfig, redirectURL string, tokenType string) (string, error)
	// GetAdminAuthToken returns the caller's live admin session handle, or an
	// error if the cookie is absent, unknown, expired, or revoked.
	GetAdminAuthToken(gc *gin.Context) (string, error)
	// NewAdminSession mints a server-side admin session and returns the opaque
	// cookie handle. See admin_session.go for why the cookie is a handle rather
	// than a hash of the admin secret.
	NewAdminSession() (string, error)
	// RefreshAdminSession extends a live admin session's absolute TTL.
	RefreshAdminSession(sessionID string) error
	// RevokeAdminSession invalidates an admin session server-side, so logout
	// actually ends it even for a cookie copy someone else holds.
	RevokeAdminSession(sessionID string) error
	// VerifyAdminSecret is the single throttled gate for every admin-secret
	// comparison. Returns (valid, locked); a locked caller never reaches the
	// comparison.
	VerifyAdminSecret(clientIP, candidate string) (valid bool, locked bool)
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
	// ValidateDelegatedAccessToken validates a stateless RFC 8693 delegated
	// access token presented at Authorizer's own API. Weaker than
	// ValidateAccessToken by exactly one property (no session lookup) and
	// stricter by one (audience must be this server) — see its doc comment.
	ValidateDelegatedAccessToken(gc *gin.Context, accessToken string) (map[string]interface{}, error)
	// ValidateDelegatedAccessTokenForResource is ValidateDelegatedAccessToken
	// with the accepted audience supplied by the caller rather than derived from
	// the request host. Exists so the MCP surface can accept delegated tokens
	// bound to "<url>/mcp" WITHOUT widening the first-party rule — see its doc
	// comment for the bijection that separation preserves.
	ValidateDelegatedAccessTokenForResource(gc *gin.Context, accessToken string, resource string) (map[string]interface{}, error)
	// ValidateMCPAccessToken validates an access token presented at the MCP
	// surface. Same stateful core as ValidateAccessToken, differing in exactly
	// one rule: `aud` must equal the caller-supplied canonical MCP resource URI
	// (RFC 8707 / MCP authorization), which is the audience ValidateAccessToken
	// rejects. Every other check, subject liveness included, is shared. Falls
	// back to ValidateDelegatedAccessTokenForResource for tokens carrying an
	// RFC 8693 `act` claim.
	ValidateMCPAccessToken(gc *gin.Context, accessToken string, resource string) (map[string]interface{}, error)
	// ValidateAdminToken validates session token
	ValidateBrowserSession(gc *gin.Context, encryptedSession string) (*SessionData, error)
	// ValidateJWTClaims validates jwt claims
	ValidateJWTClaims(claims jwt.MapClaims, authTokenConfig *AuthTokenConfig) (bool, error)
	// ValidateJWTTokenWithoutNonce validates jwt token without nonce
	ValidateJWTTokenWithoutNonce(claims jwt.MapClaims, authTokenConfig *AuthTokenConfig) (bool, error)
	// ValidateRefreshToken validates refresh token. expectedClientID is the
	// OAuth client presenting the token at the token endpoint — it MUST match
	// the token's "aud" claim (the client it was issued to).
	ValidateRefreshToken(gc *gin.Context, refreshToken string, expectedClientID string) (map[string]interface{}, error)
	// NotifyBackchannelLogout signs and POSTs an OIDC Back-Channel Logout
	// 1.0 logout_token to the supplied URI. Intended to be invoked from a
	// goroutine; remote HTTP failures are not surfaced beyond the local error.
	NotifyBackchannelLogout(ctx context.Context, uri string, cfg *BackchannelLogoutConfig) error
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
