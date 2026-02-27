package cassandradb

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddSessionToken adds a session token to the database
func (p *provider) AddSessionToken(ctx context.Context, token *schemas.SessionToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	if token.CreatedAt == 0 {
		token.CreatedAt = time.Now().Unix()
	}
	if token.UpdatedAt == 0 {
		token.UpdatedAt = time.Now().Unix()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, user_id, key_name, token_value, expires_at, created_at, updated_at) VALUES ('%s', '%s', '%s', '%s', %d, %d, %d)`,
		KeySpace+"."+schemas.Collections.SessionToken,
		token.ID, token.UserID, token.KeyName, token.Token, token.ExpiresAt, token.CreatedAt, token.UpdatedAt)
	return p.db.Query(query).Exec()
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	var token schemas.SessionToken
	query := fmt.Sprintf(`SELECT id, user_id, key_name, token_value, expires_at, created_at, updated_at FROM %s WHERE user_id = '%s' AND key_name = '%s' LIMIT 1 ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.SessionToken, userId, key)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&token.ID, &token.UserID, &token.KeyName, &token.Token, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.SessionToken, id)
	return p.db.Query(query).Exec()
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	// Cassandra doesn't support delete with non-primary key filter directly, so scan first
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE user_id = '%s' AND key_name = '%s' ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.SessionToken, userId, key)
	iter := p.db.Query(query).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.SessionToken, id)
		if err := p.db.Query(delQuery).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE user_id LIKE '%%%s%%' ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.SessionToken, userId)
	iter := p.db.Query(query).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.SessionToken, id)
		if err := p.db.Query(delQuery).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	prefix := namespace + ":"
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE user_id LIKE '%s%%' ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.SessionToken, prefix)
	iter := p.db.Query(query).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.SessionToken, id)
		if err := p.db.Query(delQuery).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE expires_at < %d ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.SessionToken, currentTime)
	iter := p.db.Query(query).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.SessionToken, id)
		if err := p.db.Query(delQuery).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	var tokens []*schemas.SessionToken
	query := fmt.Sprintf(`SELECT id, user_id, key_name, token_value, expires_at, created_at, updated_at FROM %s`,
		KeySpace+"."+schemas.Collections.SessionToken)
	iter := p.db.Query(query).Iter()
	for {
		var token schemas.SessionToken
		if !iter.Scan(&token.ID, &token.UserID, &token.KeyName, &token.Token, &token.ExpiresAt, &token.CreatedAt, &token.UpdatedAt) {
			break
		}
		tokens = append(tokens, &token)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return tokens, nil
}

// AddMFASession adds an MFA session to the database
func (p *provider) AddMFASession(ctx context.Context, session *schemas.MFASession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.CreatedAt == 0 {
		session.CreatedAt = time.Now().Unix()
	}
	if session.UpdatedAt == 0 {
		session.UpdatedAt = time.Now().Unix()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, user_id, key_name, expires_at, created_at, updated_at) VALUES ('%s', '%s', '%s', %d, %d, %d)`,
		KeySpace+"."+schemas.Collections.MFASession,
		session.ID, session.UserID, session.KeyName, session.ExpiresAt, session.CreatedAt, session.UpdatedAt)
	return p.db.Query(query).Exec()
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	var session schemas.MFASession
	query := fmt.Sprintf(`SELECT id, user_id, key_name, expires_at, created_at, updated_at FROM %s WHERE user_id = '%s' AND key_name = '%s' LIMIT 1 ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.MFASession, userId, key)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&session.ID, &session.UserID, &session.KeyName, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteMFASession deletes an MFA session by ID
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.MFASession, id)
	return p.db.Query(query).Exec()
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE user_id = '%s' AND key_name = '%s' ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.MFASession, userId, key)
	iter := p.db.Query(query).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.MFASession, id)
		if err := p.db.Query(delQuery).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	query := fmt.Sprintf(`SELECT id, user_id, key_name, expires_at, created_at, updated_at FROM %s WHERE user_id = '%s' ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.MFASession, userId)
	iter := p.db.Query(query).Iter()
	for {
		var session schemas.MFASession
		if !iter.Scan(&session.ID, &session.UserID, &session.KeyName, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt) {
			break
		}
		sessions = append(sessions, &session)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// CleanExpiredMFASessions removes expired MFA sessions from the database
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	currentTime := time.Now().Unix()
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE expires_at < %d ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.MFASession, currentTime)
	iter := p.db.Query(query).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.MFASession, id)
		if err := p.db.Query(delQuery).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetAllMFASessions retrieves all MFA sessions (for testing)
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	query := fmt.Sprintf(`SELECT id, user_id, key_name, expires_at, created_at, updated_at FROM %s`,
		KeySpace+"."+schemas.Collections.MFASession)
	iter := p.db.Query(query).Iter()
	for {
		var session schemas.MFASession
		if !iter.Scan(&session.ID, &session.UserID, &session.KeyName, &session.ExpiresAt, &session.CreatedAt, &session.UpdatedAt) {
			break
		}
		sessions = append(sessions, &session)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// AddOAuthState adds an OAuth state to the database (upsert by state_key)
func (p *provider) AddOAuthState(ctx context.Context, state *schemas.OAuthState) error {
	if state.ID == "" {
		state.ID = uuid.New().String()
	}
	if state.CreatedAt == 0 {
		state.CreatedAt = time.Now().Unix()
	}
	if state.UpdatedAt == 0 {
		state.UpdatedAt = time.Now().Unix()
	}
	// Delete existing state with same state_key first
	var existingIDs []string
	selectQuery := fmt.Sprintf(`SELECT id FROM %s WHERE state_key = '%s' ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.OAuthState, state.StateKey)
	iter := p.db.Query(selectQuery).Iter()
	var id string
	for iter.Scan(&id) {
		existingIDs = append(existingIDs, id)
	}
	iter.Close()
	for _, eid := range existingIDs {
		p.db.Query(fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.OAuthState, eid)).Exec()
	}
	query := fmt.Sprintf(`INSERT INTO %s (id, state_key, state, created_at, updated_at) VALUES ('%s', '%s', '%s', %d, %d)`,
		KeySpace+"."+schemas.Collections.OAuthState,
		state.ID, state.StateKey, state.State, state.CreatedAt, state.UpdatedAt)
	return p.db.Query(query).Exec()
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	var state schemas.OAuthState
	query := fmt.Sprintf(`SELECT id, state_key, state, created_at, updated_at FROM %s WHERE state_key = '%s' LIMIT 1 ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.OAuthState, key)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&state.ID, &state.StateKey, &state.State, &state.CreatedAt, &state.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// DeleteOAuthStateByKey deletes an OAuth state by key
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	var ids []string
	query := fmt.Sprintf(`SELECT id FROM %s WHERE state_key = '%s' ALLOW FILTERING`,
		KeySpace+"."+schemas.Collections.OAuthState, key)
	iter := p.db.Query(query).Iter()
	var id string
	for iter.Scan(&id) {
		ids = append(ids, id)
	}
	if err := iter.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		delQuery := fmt.Sprintf("DELETE FROM %s WHERE id = '%s'", KeySpace+"."+schemas.Collections.OAuthState, id)
		if err := p.db.Query(delQuery).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	var states []*schemas.OAuthState
	query := fmt.Sprintf(`SELECT id, state_key, state, created_at, updated_at FROM %s`,
		KeySpace+"."+schemas.Collections.OAuthState)
	iter := p.db.Query(query).Iter()
	for {
		var state schemas.OAuthState
		if !iter.Scan(&state.ID, &state.StateKey, &state.State, &state.CreatedAt, &state.UpdatedAt) {
			break
		}
		states = append(states, &state)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return states, nil
}
