package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies struct for db store provider
type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
}

type provider struct {
	config          *config.Config
	dependencies    *Dependencies
	storageProvider storage.Provider
}

// NewDBProvider returns a new database-backed memory store provider
func NewDBProvider(cfg *config.Config, deps *Dependencies) (*provider, error) {
	if deps.StorageProvider == nil {
		return nil, fmt.Errorf("storage provider is required for database-backed memory store")
	}
	return &provider{
		config:          cfg,
		dependencies:    deps,
		storageProvider: deps.StorageProvider,
	}, nil
}

// SetUserSession sets the user session for given user identifier in form recipe:user_id
func (p *provider) SetUserSession(userId, key, token string, expiration int64) error {
	ctx := context.Background()
	sessionToken := &schemas.SessionToken{
		ID:        uuid.New().String(),
		UserID:    userId,
		KeyName:   key,
		Token:     token,
		ExpiresAt: expiration,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	sessionToken.Key = sessionToken.ID

	// Delete expired entries first
	err := p.cleanExpiredSessionTokens(ctx)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error cleaning expired session tokens")
	}

	// Use upsert - delete existing if any, then create new
	err = p.deleteSessionTokenByUserIDAndKey(ctx, userId, key)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error deleting existing session token")
		// Continue anyway
	}

	err = p.addSessionToken(ctx, sessionToken)
	if err != nil {
		return fmt.Errorf("error setting user session: %w", err)
	}
	return nil
}

// GetUserSession returns the session token for given token
func (p *provider) GetUserSession(userId, key string) (string, error) {
	ctx := context.Background()

	// Clean expired entries first
	err := p.cleanExpiredSessionTokens(ctx)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error cleaning expired session tokens")
	}

	token, err := p.getSessionTokenByUserIDAndKey(ctx, userId, key)
	if err != nil {
		return "", fmt.Errorf("not found")
	}

	// Check expiration
	currentTime := time.Now().Unix()
	if token.ExpiresAt < currentTime {
		// Delete expired token
		_ = p.deleteSessionToken(ctx, token.ID)
		return "", fmt.Errorf("not found")
	}

	return token.Token, nil
}

// DeleteUserSession deletes the user session
func (p *provider) DeleteUserSession(userId, key string) error {
	ctx := context.Background()
	keys := []string{
		constants.TokenTypeSessionToken + "_" + key,
		constants.TokenTypeAccessToken + "_" + key,
		constants.TokenTypeRefreshToken + "_" + key,
	}

	for _, k := range keys {
		err := p.deleteSessionTokenByUserIDAndKey(ctx, userId, k)
		if err != nil {
			p.dependencies.Log.Debug().Err(err).Msgf("Error deleting session token for user %s and key %s", userId, k)
			// Continue
		}
	}
	return nil
}

// DeleteAllUserSessions deletes all the sessions from the session store
func (p *provider) DeleteAllUserSessions(userId string) error {
	ctx := context.Background()
	return p.deleteAllSessionTokensByUserID(ctx, userId)
}

// DeleteSessionForNamespace deletes the session for a given namespace
func (p *provider) DeleteSessionForNamespace(namespace string) error {
	ctx := context.Background()
	return p.deleteSessionTokensByNamespace(ctx, namespace)
}

// SetMfaSession sets the mfa session with key and value of userId
func (p *provider) SetMfaSession(userId, key string, expiration int64) error {
	ctx := context.Background()
	mfaSession := &schemas.MFASession{
		ID:        uuid.New().String(),
		UserID:    userId,
		KeyName:   key,
		ExpiresAt: expiration,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	mfaSession.Key = mfaSession.ID

	// Delete expired entries first
	err := p.cleanExpiredMFASessions(ctx)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error cleaning expired MFA sessions")
	}

	// Delete existing if any
	err = p.deleteMFASessionByUserIDAndKey(ctx, userId, key)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error deleting existing MFA session")
		// Continue anyway
	}

	err = p.addMFASession(ctx, mfaSession)
	if err != nil {
		return fmt.Errorf("error setting MFA session: %w", err)
	}
	return nil
}

// GetMfaSession returns value of given mfa session
func (p *provider) GetMfaSession(userId, key string) (string, error) {
	ctx := context.Background()

	// Clean expired entries first
	err := p.cleanExpiredMFASessions(ctx)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error cleaning expired MFA sessions")
	}

	mfaSession, err := p.getMFASessionByUserIDAndKey(ctx, userId, key)
	if err != nil {
		return "", fmt.Errorf("not found")
	}

	// Check expiration
	currentTime := time.Now().Unix()
	if mfaSession.ExpiresAt < currentTime {
		// Delete expired session
		_ = p.deleteMFASession(ctx, mfaSession.ID)
		return "", fmt.Errorf("not found")
	}

	return mfaSession.UserID, nil
}

// GetAllMfaSessions returns all mfa sessions for given userId
func (p *provider) GetAllMfaSessions(userId string) ([]string, error) {
	ctx := context.Background()

	// Clean expired entries first
	err := p.cleanExpiredMFASessions(ctx)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error cleaning expired MFA sessions")
	}

	sessions, err := p.getAllMFASessionsByUserID(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("not found")
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("not found")
	}

	keys := make([]string, 0, len(sessions))
	for _, session := range sessions {
		keys = append(keys, session.KeyName)
	}
	return keys, nil
}

// DeleteMfaSession deletes given mfa session from in-memory store.
func (p *provider) DeleteMfaSession(userId, key string) error {
	ctx := context.Background()
	return p.deleteMFASessionByUserIDAndKey(ctx, userId, key)
}

// SetState sets the login state (key, value form) in the session store
func (p *provider) SetState(key, state string) error {
	ctx := context.Background()
	oauthState := &schemas.OAuthState{
		ID:        uuid.New().String(),
		StateKey:  key,
		State:     state,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	oauthState.Key = oauthState.ID

	// Delete existing if any
	err := p.deleteOAuthStateByKey(ctx, key)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error deleting existing OAuth state")
		// Continue anyway
	}

	err = p.addOAuthState(ctx, oauthState)
	if err != nil {
		return fmt.Errorf("error setting state: %w", err)
	}
	return nil
}

// GetState returns the state from the session store
func (p *provider) GetState(key string) (string, error) {
	ctx := context.Background()
	oauthState, err := p.getOAuthStateByKey(ctx, key)
	if err != nil {
		return "", fmt.Errorf("not found")
	}
	return oauthState.State, nil
}

// RemoveState removes the social login state from the session store
func (p *provider) RemoveState(key string) error {
	ctx := context.Background()
	return p.deleteOAuthStateByKey(ctx, key)
}

// GetAllData returns all the data from the session store
// This is used for testing purposes only
func (p *provider) GetAllData() (map[string]string, error) {
	ctx := context.Background()
	data := make(map[string]string)

	// Get all session tokens
	sessionTokens, err := p.getAllSessionTokens(ctx)
	if err == nil {
		for _, token := range sessionTokens {
			key := fmt.Sprintf("%s:%s", token.UserID, token.KeyName)
			data[key] = token.Token
		}
	}

	// Get all MFA sessions
	mfaSessions, err := p.getAllMFASessions(ctx)
	if err == nil {
		for _, session := range mfaSessions {
			key := fmt.Sprintf("mfa_session_%s:%s", session.UserID, session.KeyName)
			data[key] = session.UserID
		}
	}

	// Get all OAuth states
	oauthStates, err := p.getAllOAuthStates(ctx)
	if err == nil {
		for _, state := range oauthStates {
			key := fmt.Sprintf("authorizer_state:%s", state.StateKey)
			data[key] = state.State
		}
	}

	return data, nil
}

// Helper methods for database operations

func (p *provider) addSessionToken(ctx context.Context, token *schemas.SessionToken) error {
	// This will be implemented per database type
	return p.storageProvider.AddSessionToken(ctx, token)
}

func (p *provider) getSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	return p.storageProvider.GetSessionTokenByUserIDAndKey(ctx, userId, key)
}

func (p *provider) deleteSessionToken(ctx context.Context, id string) error {
	return p.storageProvider.DeleteSessionToken(ctx, id)
}

func (p *provider) deleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	return p.storageProvider.DeleteSessionTokenByUserIDAndKey(ctx, userId, key)
}

func (p *provider) deleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	return p.storageProvider.DeleteAllSessionTokensByUserID(ctx, userId)
}

func (p *provider) deleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	return p.storageProvider.DeleteSessionTokensByNamespace(ctx, namespace)
}

func (p *provider) cleanExpiredSessionTokens(ctx context.Context) error {
	return p.storageProvider.CleanExpiredSessionTokens(ctx)
}

func (p *provider) getAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	return p.storageProvider.GetAllSessionTokens(ctx)
}

func (p *provider) addMFASession(ctx context.Context, session *schemas.MFASession) error {
	return p.storageProvider.AddMFASession(ctx, session)
}

func (p *provider) getMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	return p.storageProvider.GetMFASessionByUserIDAndKey(ctx, userId, key)
}

func (p *provider) deleteMFASession(ctx context.Context, id string) error {
	return p.storageProvider.DeleteMFASession(ctx, id)
}

func (p *provider) deleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	return p.storageProvider.DeleteMFASessionByUserIDAndKey(ctx, userId, key)
}

func (p *provider) getAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	return p.storageProvider.GetAllMFASessionsByUserID(ctx, userId)
}

func (p *provider) cleanExpiredMFASessions(ctx context.Context) error {
	return p.storageProvider.CleanExpiredMFASessions(ctx)
}

func (p *provider) getAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	return p.storageProvider.GetAllMFASessions(ctx)
}

func (p *provider) addOAuthState(ctx context.Context, state *schemas.OAuthState) error {
	return p.storageProvider.AddOAuthState(ctx, state)
}

func (p *provider) getOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	return p.storageProvider.GetOAuthStateByKey(ctx, key)
}

func (p *provider) deleteOAuthStateByKey(ctx context.Context, key string) error {
	return p.storageProvider.DeleteOAuthStateByKey(ctx, key)
}

func (p *provider) getAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	return p.storageProvider.GetAllOAuthStates(ctx)
}
