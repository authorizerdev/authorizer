package couchbase

import (
	"context"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
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
	_, err := p.db.Collection(schemas.Collections.SessionToken).Insert(token.ID, token, &gocb.InsertOptions{Context: ctx})
	return err
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	var token schemas.SessionToken
	query := fmt.Sprintf(`SELECT _id, user_id, key_name, token, expires_at, created_at, updated_at FROM %s.%s WHERE user_id = $1 AND key_name = $2 LIMIT 1`,
		p.scopeName, schemas.Collections.SessionToken)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{userId, key},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	_, err := p.db.Collection(schemas.Collections.SessionToken).Remove(id, &gocb.RemoveOptions{Context: ctx})
	return err
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	query := fmt.Sprintf(`SELECT _id FROM %s.%s WHERE user_id = $1 AND key_name = $2`,
		p.scopeName, schemas.Collections.SessionToken)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{userId, key},
	})
	if err != nil {
		return err
	}
	type idRow struct {
		ID string `json:"_id"`
	}
	for q.Next() {
		var row idRow
		if err := q.Row(&row); err != nil {
			continue
		}
		p.db.Collection(schemas.Collections.SessionToken).Remove(row.ID, &gocb.RemoveOptions{Context: ctx})
	}
	return nil
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	query := fmt.Sprintf(`SELECT _id FROM %s.%s WHERE CONTAINS(user_id, $1)`,
		p.scopeName, schemas.Collections.SessionToken)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{userId},
	})
	if err != nil {
		return err
	}
	type idRow struct {
		ID string `json:"_id"`
	}
	for q.Next() {
		var row idRow
		if err := q.Row(&row); err != nil {
			continue
		}
		p.db.Collection(schemas.Collections.SessionToken).Remove(row.ID, &gocb.RemoveOptions{Context: ctx})
	}
	return nil
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	prefix := namespace + ":"
	query := fmt.Sprintf(`SELECT _id FROM %s.%s WHERE STARTS_WITH(user_id, $1)`,
		p.scopeName, schemas.Collections.SessionToken)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{prefix},
	})
	if err != nil {
		return err
	}
	type idRow struct {
		ID string `json:"_id"`
	}
	for q.Next() {
		var row idRow
		if err := q.Row(&row); err != nil {
			continue
		}
		p.db.Collection(schemas.Collections.SessionToken).Remove(row.ID, &gocb.RemoveOptions{Context: ctx})
	}
	return nil
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	query := fmt.Sprintf(`SELECT _id FROM %s.%s WHERE expires_at < $1`,
		p.scopeName, schemas.Collections.SessionToken)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{currentTime},
	})
	if err != nil {
		return err
	}
	type idRow struct {
		ID string `json:"_id"`
	}
	for q.Next() {
		var row idRow
		if err := q.Row(&row); err != nil {
			continue
		}
		p.db.Collection(schemas.Collections.SessionToken).Remove(row.ID, &gocb.RemoveOptions{Context: ctx})
	}
	return nil
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	var tokens []*schemas.SessionToken
	query := fmt.Sprintf(`SELECT _id, user_id, key_name, token, expires_at, created_at, updated_at FROM %s.%s`,
		p.scopeName, schemas.Collections.SessionToken)
	q, err := p.db.Query(query, &gocb.QueryOptions{ScanConsistency: gocb.QueryScanConsistencyRequestPlus})
	if err != nil {
		return nil, err
	}
	for q.Next() {
		var token schemas.SessionToken
		if err := q.Row(&token); err != nil {
			continue
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
	if session.CreatedAt == 0 {
		session.CreatedAt = time.Now().Unix()
	}
	if session.UpdatedAt == 0 {
		session.UpdatedAt = time.Now().Unix()
	}
	_, err := p.db.Collection(schemas.Collections.MFASession).Insert(session.ID, session, &gocb.InsertOptions{Context: ctx})
	return err
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	var session schemas.MFASession
	query := fmt.Sprintf(`SELECT _id, user_id, key_name, expires_at, created_at, updated_at FROM %s.%s WHERE user_id = $1 AND key_name = $2 LIMIT 1`,
		p.scopeName, schemas.Collections.MFASession)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{userId, key},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteMFASession deletes an MFA session by ID
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	_, err := p.db.Collection(schemas.Collections.MFASession).Remove(id, &gocb.RemoveOptions{Context: ctx})
	return err
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	query := fmt.Sprintf(`SELECT _id FROM %s.%s WHERE user_id = $1 AND key_name = $2`,
		p.scopeName, schemas.Collections.MFASession)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{userId, key},
	})
	if err != nil {
		return err
	}
	type idRow struct {
		ID string `json:"_id"`
	}
	for q.Next() {
		var row idRow
		if err := q.Row(&row); err != nil {
			continue
		}
		p.db.Collection(schemas.Collections.MFASession).Remove(row.ID, &gocb.RemoveOptions{Context: ctx})
	}
	return nil
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	query := fmt.Sprintf(`SELECT _id, user_id, key_name, expires_at, created_at, updated_at FROM %s.%s WHERE user_id = $1`,
		p.scopeName, schemas.Collections.MFASession)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{userId},
	})
	if err != nil {
		return nil, err
	}
	for q.Next() {
		var session schemas.MFASession
		if err := q.Row(&session); err != nil {
			continue
		}
		sessions = append(sessions, &session)
	}
	return sessions, nil
}

// CleanExpiredMFASessions removes expired MFA sessions from the database
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	currentTime := time.Now().Unix()
	query := fmt.Sprintf(`SELECT _id FROM %s.%s WHERE expires_at < $1`,
		p.scopeName, schemas.Collections.MFASession)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{currentTime},
	})
	if err != nil {
		return err
	}
	type idRow struct {
		ID string `json:"_id"`
	}
	for q.Next() {
		var row idRow
		if err := q.Row(&row); err != nil {
			continue
		}
		p.db.Collection(schemas.Collections.MFASession).Remove(row.ID, &gocb.RemoveOptions{Context: ctx})
	}
	return nil
}

// GetAllMFASessions retrieves all MFA sessions (for testing)
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	var sessions []*schemas.MFASession
	query := fmt.Sprintf(`SELECT _id, user_id, key_name, expires_at, created_at, updated_at FROM %s.%s`,
		p.scopeName, schemas.Collections.MFASession)
	q, err := p.db.Query(query, &gocb.QueryOptions{ScanConsistency: gocb.QueryScanConsistencyRequestPlus})
	if err != nil {
		return nil, err
	}
	for q.Next() {
		var session schemas.MFASession
		if err := q.Row(&session); err != nil {
			continue
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
	if state.CreatedAt == 0 {
		state.CreatedAt = time.Now().Unix()
	}
	if state.UpdatedAt == 0 {
		state.UpdatedAt = time.Now().Unix()
	}
	// Delete existing state with same state_key first (upsert behavior)
	delQuery := fmt.Sprintf(`SELECT _id FROM %s.%s WHERE state_key = $1`, p.scopeName, schemas.Collections.OAuthState)
	q, _ := p.db.Query(delQuery, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{state.StateKey},
	})
	if q != nil {
		type idRow struct {
			ID string `json:"_id"`
		}
		for q.Next() {
			var row idRow
			if err := q.Row(&row); err != nil {
				continue
			}
			p.db.Collection(schemas.Collections.OAuthState).Remove(row.ID, &gocb.RemoveOptions{Context: ctx})
		}
	}
	_, err := p.db.Collection(schemas.Collections.OAuthState).Insert(state.ID, state, &gocb.InsertOptions{Context: ctx})
	return err
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	var state schemas.OAuthState
	query := fmt.Sprintf(`SELECT _id, state_key, state, created_at, updated_at FROM %s.%s WHERE state_key = $1 LIMIT 1`,
		p.scopeName, schemas.Collections.OAuthState)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{key},
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// DeleteOAuthStateByKey deletes an OAuth state by key
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	query := fmt.Sprintf(`SELECT _id FROM %s.%s WHERE state_key = $1`,
		p.scopeName, schemas.Collections.OAuthState)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		ScanConsistency:      gocb.QueryScanConsistencyRequestPlus,
		PositionalParameters: []interface{}{key},
	})
	if err != nil {
		return err
	}
	type idRow struct {
		ID string `json:"_id"`
	}
	for q.Next() {
		var row idRow
		if err := q.Row(&row); err != nil {
			continue
		}
		p.db.Collection(schemas.Collections.OAuthState).Remove(row.ID, &gocb.RemoveOptions{Context: ctx})
	}
	return nil
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	var states []*schemas.OAuthState
	query := fmt.Sprintf(`SELECT _id, state_key, state, created_at, updated_at FROM %s.%s`,
		p.scopeName, schemas.Collections.OAuthState)
	q, err := p.db.Query(query, &gocb.QueryOptions{ScanConsistency: gocb.QueryScanConsistencyRequestPlus})
	if err != nil {
		return nil, err
	}
	for q.Next() {
		var state schemas.OAuthState
		if err := q.Row(&state); err != nil {
			continue
		}
		states = append(states, &state)
	}
	return states, nil
}
