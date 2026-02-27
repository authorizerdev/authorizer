package provider_template

import (
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Dependencies struct the TODO(replace with new db name) data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

// TODO change following provider to new db provider
type provider struct {
	config       *config.Config
	dependencies *Dependencies
	db           *gorm.DB
}

// NewProvider returns a new provider for your database type.
// TODO: change provider struct and NewProvider to use your database client.
//
// This provider must implement all methods from storage.Provider, including:
// - User, VerificationRequest, Session, Webhook, EmailTemplate, OTP, Authenticator
// - Memory store methods (when Redis is not configured):
//   - SessionToken: AddSessionToken, GetSessionTokenByUserIDAndKey, DeleteSessionToken,
//     DeleteSessionTokenByUserIDAndKey, DeleteAllSessionTokensByUserID,
//     DeleteSessionTokensByNamespace, CleanExpiredSessionTokens, GetAllSessionTokens
//   - MFASession: AddMFASession, GetMFASessionByUserIDAndKey, DeleteMFASession,
//     DeleteMFASessionByUserIDAndKey, GetAllMFASessionsByUserID,
//     CleanExpiredMFASessions, GetAllMFASessions
//   - OAuthState: AddOAuthState, GetOAuthStateByKey, DeleteOAuthStateByKey, GetAllOAuthStates
//
// Use schemas.Collections for table/collection names (e.g., schemas.Collections.SessionToken).
func NewProvider(
	config *config.Config,
	deps *Dependencies,
) (*provider, error) {
	var sqlDB *gorm.DB

	return &provider{
		config:       config,
		dependencies: deps,
		db:           sqlDB,
	}, nil
}
