package dynamodb

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddSessionToken adds a session token to the database
func (p *provider) AddSessionToken(ctx context.Context, token *schemas.SessionToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	token.Key = token.ID
	if token.CreatedAt == 0 {
		token.CreatedAt = time.Now().Unix()
	}
	if token.UpdatedAt == 0 {
		token.UpdatedAt = time.Now().Unix()
	}
	collection := p.db.Table(schemas.Collections.SessionToken)
	return collection.Put(token).RunWithContext(ctx)
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	var tokens []schemas.SessionToken
	collection := p.db.Table(schemas.Collections.SessionToken)
	err := collection.Scan().
		Index("user_id").
		Filter("'user_id' = ? AND 'key_name' = ?", userId, key).
		Limit(1).
		AllWithContext(ctx, &tokens)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, errors.New("session token not found")
	}
	return &tokens[0], nil
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	collection := p.db.Table(schemas.Collections.SessionToken)
	return collection.Delete("id", id).RunWithContext(ctx)
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	var tokens []schemas.SessionToken
	collection := p.db.Table(schemas.Collections.SessionToken)
	err := collection.Scan().
		Index("user_id").
		Filter("'user_id' = ? AND 'key_name' = ?", userId, key).
		AllWithContext(ctx, &tokens)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		if err := collection.Delete("id", token.ID).RunWithContext(ctx); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	var tokens []schemas.SessionToken
	collection := p.db.Table(schemas.Collections.SessionToken)
	err := collection.Scan().AllWithContext(ctx, &tokens)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		if strings.Contains(token.UserID, userId) {
			if err := collection.Delete("id", token.ID).RunWithContext(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	prefix := namespace + ":"
	var tokens []schemas.SessionToken
	collection := p.db.Table(schemas.Collections.SessionToken)
	err := collection.Scan().AllWithContext(ctx, &tokens)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		if strings.HasPrefix(token.UserID, prefix) {
			if err := collection.Delete("id", token.ID).RunWithContext(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	var tokens []schemas.SessionToken
	collection := p.db.Table(schemas.Collections.SessionToken)
	err := collection.Scan().Filter("'expires_at' < ?", currentTime).AllWithContext(ctx, &tokens)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		if err := collection.Delete("id", token.ID).RunWithContext(ctx); err != nil {
			return err
		}
	}
	return nil
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	var tokens []schemas.SessionToken
	collection := p.db.Table(schemas.Collections.SessionToken)
	err := collection.Scan().AllWithContext(ctx, &tokens)
	if err != nil {
		return nil, err
	}
	var result []*schemas.SessionToken
	for i := range tokens {
		result = append(result, &tokens[i])
	}
	return result, nil
}

// AddMFASession adds an MFA session to the database
func (p *provider) AddMFASession(ctx context.Context, session *schemas.MFASession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	session.Key = session.ID
	if session.CreatedAt == 0 {
		session.CreatedAt = time.Now().Unix()
	}
	if session.UpdatedAt == 0 {
		session.UpdatedAt = time.Now().Unix()
	}
	collection := p.db.Table(schemas.Collections.MFASession)
	return collection.Put(session).RunWithContext(ctx)
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	var sessions []schemas.MFASession
	collection := p.db.Table(schemas.Collections.MFASession)
	err := collection.Scan().
		Index("user_id").
		Filter("'user_id' = ? AND 'key_name' = ?", userId, key).
		Limit(1).
		AllWithContext(ctx, &sessions)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, errors.New("MFA session not found")
	}
	return &sessions[0], nil
}

// DeleteMFASession deletes an MFA session by ID
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	collection := p.db.Table(schemas.Collections.MFASession)
	return collection.Delete("id", id).RunWithContext(ctx)
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	var sessions []schemas.MFASession
	collection := p.db.Table(schemas.Collections.MFASession)
	err := collection.Scan().
		Index("user_id").
		Filter("'user_id' = ? AND 'key_name' = ?", userId, key).
		AllWithContext(ctx, &sessions)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if err := collection.Delete("id", session.ID).RunWithContext(ctx); err != nil {
			return err
		}
	}
	return nil
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	var sessions []schemas.MFASession
	collection := p.db.Table(schemas.Collections.MFASession)
	err := collection.Scan().
		Index("user_id").
		Filter("'user_id' = ?", userId).
		AllWithContext(ctx, &sessions)
	if err != nil {
		return nil, err
	}
	var result []*schemas.MFASession
	for i := range sessions {
		result = append(result, &sessions[i])
	}
	return result, nil
}

// CleanExpiredMFASessions removes expired MFA sessions from the database
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	currentTime := time.Now().Unix()
	var sessions []schemas.MFASession
	collection := p.db.Table(schemas.Collections.MFASession)
	err := collection.Scan().Filter("'expires_at' < ?", currentTime).AllWithContext(ctx, &sessions)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if err := collection.Delete("id", session.ID).RunWithContext(ctx); err != nil {
			return err
		}
	}
	return nil
}

// GetAllMFASessions retrieves all MFA sessions (for testing)
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	var sessions []schemas.MFASession
	collection := p.db.Table(schemas.Collections.MFASession)
	err := collection.Scan().AllWithContext(ctx, &sessions)
	if err != nil {
		return nil, err
	}
	var result []*schemas.MFASession
	for i := range sessions {
		result = append(result, &sessions[i])
	}
	return result, nil
}

// AddOAuthState adds an OAuth state to the database (upsert by state_key)
func (p *provider) AddOAuthState(ctx context.Context, state *schemas.OAuthState) error {
	if state.ID == "" {
		state.ID = uuid.New().String()
	}
	state.Key = state.ID
	if state.CreatedAt == 0 {
		state.CreatedAt = time.Now().Unix()
	}
	if state.UpdatedAt == 0 {
		state.UpdatedAt = time.Now().Unix()
	}
	// Delete existing state with same state_key first (upsert behavior)
	var existing []schemas.OAuthState
	collection := p.db.Table(schemas.Collections.OAuthState)
	collection.Scan().
		Index("state_key").
		Filter("'state_key' = ?", state.StateKey).
		AllWithContext(ctx, &existing)
	for _, e := range existing {
		collection.Delete("id", e.ID).RunWithContext(ctx)
	}
	return collection.Put(state).RunWithContext(ctx)
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	var states []schemas.OAuthState
	collection := p.db.Table(schemas.Collections.OAuthState)
	err := collection.Scan().
		Index("state_key").
		Filter("'state_key' = ?", key).
		Limit(1).
		AllWithContext(ctx, &states)
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, errors.New("OAuth state not found")
	}
	return &states[0], nil
}

// DeleteOAuthStateByKey deletes an OAuth state by key
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	var states []schemas.OAuthState
	collection := p.db.Table(schemas.Collections.OAuthState)
	err := collection.Scan().
		Index("state_key").
		Filter("'state_key' = ?", key).
		AllWithContext(ctx, &states)
	if err != nil {
		return err
	}
	for _, state := range states {
		if err := collection.Delete("id", state.ID).RunWithContext(ctx); err != nil {
			return err
		}
	}
	return nil
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	var states []schemas.OAuthState
	collection := p.db.Table(schemas.Collections.OAuthState)
	err := collection.Scan().AllWithContext(ctx, &states)
	if err != nil {
		return nil, err
	}
	var result []*schemas.OAuthState
	for i := range states {
		result = append(result, &states[i])
	}
	return result, nil
}
