package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// mfaPurposeSeparator joins the session key and its purpose inside the
// persisted KeyName. The MFASession schema has no dedicated purpose column and
// adding one would touch every DB provider (cassandra uses an explicit column
// list), so the purpose rides along in KeyName as "<key>::<purpose>". The key
// is a UUID, which never contains "::", so the split is unambiguous.
const mfaPurposeSeparator = "::"

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

	// Check expiration. Inclusive (<=): a token is invalid AT its expiry second,
	// not one second later. This matches the in-memory store, which serves an
	// entry only while ExpiresAt > now, and Redis, whose native TTL drops the key
	// at the deadline. The exclusive form left a session usable for one extra
	// second on this provider only — a cross-provider parity gap.
	currentTime := time.Now().Unix()
	if token.ExpiresAt <= currentTime {
		// Delete expired token
		_ = p.deleteSessionToken(ctx, token.ID)
		return "", fmt.Errorf("not found")
	}

	return token.Token, nil
}

// ClaimRefreshToken atomically consumes the refresh-token entry for this nonce
// and reports whether this call consumed it (see memory_store.Provider). The
// storage layer decides the race in one operation — this method must never
// read-then-delete.
func (p *provider) ClaimRefreshToken(userId, nonce string) (bool, error) {
	ctx := context.Background()
	return p.storageProvider.DeleteSessionTokenByUserIDAndKey(ctx, userId, constants.TokenTypeRefreshToken+"_"+nonce)
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

// SetMfaSession sets the mfa session, storing purpose in the persisted KeyName
// alongside the key (see mfaPurposeSeparator).
func (p *provider) SetMfaSession(userId, key, purpose string, expiration int64) error {
	ctx := context.Background()
	storedKey := key + mfaPurposeSeparator + purpose
	mfaSession := &schemas.MFASession{
		ID:        uuid.New().String(),
		UserID:    userId,
		KeyName:   storedKey,
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
	err = p.deleteMFASessionByUserIDAndKey(ctx, userId, storedKey)
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

// GetMfaSession returns the stored purpose of the given mfa session. The key is
// looked up against the "<key>::<purpose>" prefix of the persisted KeyName.
func (p *provider) GetMfaSession(userId, key string) (string, error) {
	ctx := context.Background()

	// Clean expired entries first
	err := p.cleanExpiredMFASessions(ctx)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error cleaning expired MFA sessions")
	}

	sessions, err := p.getAllMFASessionsByUserID(ctx, userId)
	if err != nil {
		return "", fmt.Errorf("not found")
	}

	prefix := key + mfaPurposeSeparator
	currentTime := time.Now().Unix()
	for _, session := range sessions {
		if !strings.HasPrefix(session.KeyName, prefix) {
			continue
		}
		// Inclusive expiry, matching GetUserSession and the in-memory store.
		if session.ExpiresAt <= currentTime {
			_ = p.deleteMFASession(ctx, session.ID)
			return "", fmt.Errorf("not found")
		}
		return strings.TrimPrefix(session.KeyName, prefix), nil
	}
	return "", fmt.Errorf("not found")
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
		k := session.KeyName
		if idx := strings.Index(k, mfaPurposeSeparator); idx >= 0 {
			k = k[:idx]
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// DeleteMfaSession deletes given mfa session from the store. KeyName carries a
// "<key>::<purpose>" suffix, so match by prefix rather than exact key.
func (p *provider) DeleteMfaSession(userId, key string) error {
	ctx := context.Background()
	sessions, err := p.getAllMFASessionsByUserID(ctx, userId)
	if err != nil {
		return nil
	}
	prefix := key + mfaPurposeSeparator
	for _, session := range sessions {
		if strings.HasPrefix(session.KeyName, prefix) {
			_ = p.deleteMFASession(ctx, session.ID)
		}
	}
	return nil
}

// GetMfaSessionOwner resolves the userID and purpose for a bare mfa session
// key, without knowing the owning userID. The persisted KeyName carries a
// "<key>::<purpose>" form, so match on the "<key>::" prefix.
func (p *provider) GetMfaSessionOwner(key string) (string, string, error) {
	ctx := context.Background()

	// Clean expired entries first
	err := p.cleanExpiredMFASessions(ctx)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error cleaning expired MFA sessions")
	}

	sessions, err := p.getAllMFASessions(ctx)
	if err != nil {
		return "", "", fmt.Errorf("not found")
	}

	prefix := key + mfaPurposeSeparator
	for _, session := range sessions {
		if strings.HasPrefix(session.KeyName, prefix) {
			return session.UserID, strings.TrimPrefix(session.KeyName, prefix), nil
		}
	}
	return "", "", fmt.Errorf("not found")
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

	// Delete existing if any. Whether a row was actually removed is irrelevant
	// here — this is an overwrite, not a single-use claim.
	if _, err := p.deleteOAuthStateByKey(ctx, key); err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Error deleting existing OAuth state")
		// Continue anyway
	}

	err := p.addOAuthState(ctx, oauthState)
	if err != nil {
		return fmt.Errorf("error setting state: %w", err)
	}
	return nil
}

// GetState returns the state from the session store.
// RFC 6749 §4.1.2: authorization codes (and associated state) MUST be
// short-lived. Entries older than 10 minutes are treated as expired.
func (p *provider) GetState(key string) (string, error) {
	ctx := context.Background()
	oauthState, err := p.getOAuthStateByKey(ctx, key)
	if err != nil {
		return "", fmt.Errorf("not found")
	}
	// Enforce 10-minute TTL consistent with Redis provider.
	if oauthState.CreatedAt > 0 && time.Now().Unix()-oauthState.CreatedAt > 600 {
		// Clean up expired entry asynchronously.
		go func() { _, _ = p.deleteOAuthStateByKey(context.Background(), key) }()
		return "", fmt.Errorf("state expired")
	}
	return oauthState.State, nil
}

// RemoveState removes the social login state from the session store. Unlike
// GetAndRemoveState this is an unconditional cleanup, so "was it already gone"
// is not an error.
func (p *provider) RemoveState(key string) error {
	ctx := context.Background()
	_, err := p.deleteOAuthStateByKey(ctx, key)
	return err
}

// GetAndRemoveState atomically retrieves and deletes the state from the DB.
// Enforces 10-minute TTL consistent with other providers.
//
// The DELETE — not the preceding read — is what makes this single-use. The read
// only fetches the payload and can legitimately succeed for several concurrent
// callers redeeming the same authorization code; the storage layer's delete
// decides, in one atomic operation, which single caller actually consumed it
// (see storage.Provider.DeleteOAuthStateByKey). Returning the state on the
// strength of the read alone would hand the same code to every racer, which is
// an authorization-code replay (RFC 6749 §4.1.2).
func (p *provider) GetAndRemoveState(key string) (string, error) {
	ctx := context.Background()
	oauthState, err := p.getOAuthStateByKey(ctx, key)
	if err != nil {
		return "", fmt.Errorf("not found")
	}
	claimed, err := p.deleteOAuthStateByKey(ctx, key)
	if err != nil {
		return "", fmt.Errorf("not found")
	}
	if !claimed {
		// Another caller won the delete: this request is redeeming an already
		// consumed key. Same opaque error as a missing key — no replay oracle.
		return "", fmt.Errorf("not found")
	}
	// Enforce 10-minute TTL.
	if oauthState.CreatedAt > 0 && time.Now().Unix()-oauthState.CreatedAt > 600 {
		return "", fmt.Errorf("state expired")
	}
	return oauthState.State, nil
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

// deleteSessionTokenByUserIDAndKey removes an entry, discarding the storage
// layer's "did I claim it" answer — the callers here (upsert, logout-style
// cleanup) only need it gone. Use ClaimRefreshToken where the answer matters.
func (p *provider) deleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	_, err := p.storageProvider.DeleteSessionTokenByUserIDAndKey(ctx, userId, key)
	return err
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

// deleteOAuthStateByKey removes the entry and reports whether THIS call was the
// one that removed it (see storage.Provider.DeleteOAuthStateByKey).
func (p *provider) deleteOAuthStateByKey(ctx context.Context, key string) (bool, error) {
	return p.storageProvider.DeleteOAuthStateByKey(ctx, key)
}

func (p *provider) getAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	return p.storageProvider.GetAllOAuthStates(ctx)
}
