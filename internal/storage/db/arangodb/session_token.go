package arangodb

import (
	"context"
	"fmt"
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
	collection, err := p.db.Collection(ctx, schemas.Collections.SessionToken)
	if err != nil {
		return err
	}
	meta, err := collection.CreateDocument(ctx, token)
	if err != nil {
		return err
	}
	token.Key = meta.Key
	token.ID = meta.ID.String()
	return nil
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	var token schemas.SessionToken
	query := fmt.Sprintf("FOR d IN %s FILTER d.user_id == @user_id AND d.key_name == @key_name RETURN d", schemas.Collections.SessionToken)
	bindVars := map[string]interface{}{
		"user_id":  userId,
		"key_name": key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	if cursor.HasMore() {
		_, err = cursor.ReadDocument(ctx, &token)
		if err != nil {
			return nil, err
		}
		return &token, nil
	}
	return nil, fmt.Errorf("session token not found")
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d._id == @id REMOVE d IN %s", schemas.Collections.SessionToken, schemas.Collections.SessionToken)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d.user_id == @user_id AND d.key_name == @key_name REMOVE d IN %s", schemas.Collections.SessionToken, schemas.Collections.SessionToken)
	bindVars := map[string]interface{}{
		"user_id":  userId,
		"key_name": key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER CONTAINS(d.user_id, @user_id) REMOVE d IN %s", schemas.Collections.SessionToken, schemas.Collections.SessionToken)
	bindVars := map[string]interface{}{
		"user_id": userId,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	prefix := namespace + ":"
	query := fmt.Sprintf("FOR d IN %s FILTER STARTS_WITH(d.user_id, @prefix) REMOVE d IN %s", schemas.Collections.SessionToken, schemas.Collections.SessionToken)
	bindVars := map[string]interface{}{
		"prefix": prefix,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	query := fmt.Sprintf("FOR d IN %s FILTER d.expires_at < @current_time REMOVE d IN %s", schemas.Collections.SessionToken, schemas.Collections.SessionToken)
	bindVars := map[string]interface{}{
		"current_time": currentTime,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	var tokens []*schemas.SessionToken
	query := fmt.Sprintf("FOR d IN %s RETURN d", schemas.Collections.SessionToken)
	cursor, err := p.db.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for cursor.HasMore() {
		var token schemas.SessionToken
		_, err = cursor.ReadDocument(ctx, &token)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, &token)
	}
	return tokens, nil
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
	collection, err := p.db.Collection(ctx, schemas.Collections.MFASession)
	if err != nil {
		return err
	}
	meta, err := collection.CreateDocument(ctx, session)
	if err != nil {
		return err
	}
	session.Key = meta.Key
	session.ID = meta.ID.String()
	return nil
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	var session schemas.MFASession
	query := fmt.Sprintf("FOR d IN %s FILTER d.user_id == @user_id AND d.key_name == @key_name RETURN d", schemas.Collections.MFASession)
	bindVars := map[string]interface{}{
		"user_id":  userId,
		"key_name": key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	if cursor.HasMore() {
		_, err = cursor.ReadDocument(ctx, &session)
		if err != nil {
			return nil, err
		}
		return &session, nil
	}
	return nil, fmt.Errorf("MFA session not found")
}

// DeleteMFASession deletes an MFA session by ID
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d._id == @id REMOVE d IN %s", schemas.Collections.MFASession, schemas.Collections.MFASession)
	bindVars := map[string]interface{}{
		"id": id,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d.user_id == @user_id AND d.key_name == @key_name REMOVE d IN %s", schemas.Collections.MFASession, schemas.Collections.MFASession)
	bindVars := map[string]interface{}{
		"user_id":  userId,
		"key_name": key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	query := fmt.Sprintf("FOR d IN %s FILTER d.user_id == @user_id RETURN d", schemas.Collections.MFASession)
	bindVars := map[string]interface{}{
		"user_id": userId,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for cursor.HasMore() {
		var session schemas.MFASession
		_, err = cursor.ReadDocument(ctx, &session)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}
	return sessions, nil
}

// CleanExpiredMFASessions removes expired MFA sessions from the database
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	currentTime := time.Now().Unix()
	query := fmt.Sprintf("FOR d IN %s FILTER d.expires_at < @current_time REMOVE d IN %s", schemas.Collections.MFASession, schemas.Collections.MFASession)
	bindVars := map[string]interface{}{
		"current_time": currentTime,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// GetAllMFASessions retrieves all MFA sessions (for testing)
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	query := fmt.Sprintf("FOR d IN %s RETURN d", schemas.Collections.MFASession)
	cursor, err := p.db.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for cursor.HasMore() {
		var session schemas.MFASession
		_, err = cursor.ReadDocument(ctx, &session)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, &session)
	}
	return sessions, nil
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
	// Delete existing state with the same state_key first (upsert behavior)
	deleteQuery := fmt.Sprintf("FOR d IN %s FILTER d.state_key == @state_key REMOVE d IN %s", schemas.Collections.OAuthState, schemas.Collections.OAuthState)
	deleteCursor, err := p.db.Query(ctx, deleteQuery, map[string]interface{}{"state_key": state.StateKey})
	if err != nil {
		return err
	}
	deleteCursor.Close()

	collection, err := p.db.Collection(ctx, schemas.Collections.OAuthState)
	if err != nil {
		return err
	}
	meta, err := collection.CreateDocument(ctx, state)
	if err != nil {
		return err
	}
	state.Key = meta.Key
	state.ID = meta.ID.String()
	return nil
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	var state schemas.OAuthState
	query := fmt.Sprintf("FOR d IN %s FILTER d.state_key == @state_key RETURN d", schemas.Collections.OAuthState)
	bindVars := map[string]interface{}{
		"state_key": key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	if cursor.HasMore() {
		_, err = cursor.ReadDocument(ctx, &state)
		if err != nil {
			return nil, err
		}
		return &state, nil
	}
	return nil, fmt.Errorf("OAuth state not found")
}

// DeleteOAuthStateByKey deletes an OAuth state by key
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	query := fmt.Sprintf("FOR d IN %s FILTER d.state_key == @state_key REMOVE d IN %s", schemas.Collections.OAuthState, schemas.Collections.OAuthState)
	bindVars := map[string]interface{}{
		"state_key": key,
	}
	cursor, err := p.db.Query(ctx, query, bindVars)
	if err != nil {
		return err
	}
	defer cursor.Close()
	return nil
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	var states []*schemas.OAuthState
	query := fmt.Sprintf("FOR d IN %s RETURN d", schemas.Collections.OAuthState)
	cursor, err := p.db.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for cursor.HasMore() {
		var state schemas.OAuthState
		_, err = cursor.ReadDocument(ctx, &state)
		if err != nil {
			return nil, err
		}
		states = append(states, &state)
	}
	return states, nil
}

