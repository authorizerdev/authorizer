package dynamodb

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
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
	return p.putItem(ctx, schemas.Collections.SessionToken, token)
}

// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
func (p *provider) GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error) {
	f := expression.Name("key_name").Equal(expression.Value(key))
	items, err := p.queryEqLimit(ctx, schemas.Collections.SessionToken, "user_id", "user_id", userId, &f, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("session token not found")
	}
	var t schemas.SessionToken
	if err := unmarshalItem(items[0], &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteSessionToken deletes a session token by ID
func (p *provider) DeleteSessionToken(ctx context.Context, id string) error {
	return p.deleteItemByHash(ctx, schemas.Collections.SessionToken, "id", id)
}

// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
func (p *provider) DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error {
	f := expression.Name("key_name").Equal(expression.Value(key))
	items, err := p.queryEq(ctx, schemas.Collections.SessionToken, "user_id", "user_id", userId, &f)
	if err != nil {
		return err
	}
	for _, it := range items {
		var t schemas.SessionToken
		if err := unmarshalItem(it, &t); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.SessionToken, "id", t.ID); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
func (p *provider) DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error {
	items, err := p.scanAllRaw(ctx, schemas.Collections.SessionToken, nil, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		var t schemas.SessionToken
		if err := unmarshalItem(it, &t); err != nil {
			return err
		}
		if strings.Contains(t.UserID, userId) {
			if err := p.deleteItemByHash(ctx, schemas.Collections.SessionToken, "id", t.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteSessionTokensByNamespace deletes all session tokens for a namespace
func (p *provider) DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error {
	prefix := namespace + ":"
	items, err := p.scanAllRaw(ctx, schemas.Collections.SessionToken, nil, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		var t schemas.SessionToken
		if err := unmarshalItem(it, &t); err != nil {
			return err
		}
		if strings.HasPrefix(t.UserID, prefix) {
			if err := p.deleteItemByHash(ctx, schemas.Collections.SessionToken, "id", t.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// CleanExpiredSessionTokens removes expired session tokens from the database
func (p *provider) CleanExpiredSessionTokens(ctx context.Context) error {
	currentTime := time.Now().Unix()
	f := expression.Name("expires_at").LessThan(expression.Value(currentTime))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.SessionToken, nil, &f)
	if err != nil {
		return err
	}
	for _, it := range items {
		var t schemas.SessionToken
		if err := unmarshalItem(it, &t); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.SessionToken, "id", t.ID); err != nil {
			return err
		}
	}
	return nil
}

// GetAllSessionTokens retrieves all session tokens (for testing)
func (p *provider) GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error) {
	items, err := p.scanAllRaw(ctx, schemas.Collections.SessionToken, nil, nil)
	if err != nil {
		return nil, err
	}
	var result []*schemas.SessionToken
	for _, it := range items {
		var t schemas.SessionToken
		if err := unmarshalItem(it, &t); err != nil {
			return nil, err
		}
		result = append(result, &t)
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
	return p.putItem(ctx, schemas.Collections.MFASession, session)
}

// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
func (p *provider) GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error) {
	f := expression.Name("key_name").Equal(expression.Value(key))
	items, err := p.queryEqLimit(ctx, schemas.Collections.MFASession, "user_id", "user_id", userId, &f, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("MFA session not found")
	}
	var s schemas.MFASession
	if err := unmarshalItem(items[0], &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteMFASession deletes an MFA session by ID
func (p *provider) DeleteMFASession(ctx context.Context, id string) error {
	return p.deleteItemByHash(ctx, schemas.Collections.MFASession, "id", id)
}

// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
func (p *provider) DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error {
	f := expression.Name("key_name").Equal(expression.Value(key))
	items, err := p.queryEq(ctx, schemas.Collections.MFASession, "user_id", "user_id", userId, &f)
	if err != nil {
		return err
	}
	for _, it := range items {
		var s schemas.MFASession
		if err := unmarshalItem(it, &s); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.MFASession, "id", s.ID); err != nil {
			return err
		}
	}
	return nil
}

// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
func (p *provider) GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error) {
	items, err := p.queryEq(ctx, schemas.Collections.MFASession, "user_id", "user_id", userId, nil)
	if err != nil {
		return nil, err
	}
	var result []*schemas.MFASession
	for _, it := range items {
		var s schemas.MFASession
		if err := unmarshalItem(it, &s); err != nil {
			return nil, err
		}
		result = append(result, &s)
	}
	return result, nil
}

// CleanExpiredMFASessions removes expired MFA sessions from the database
func (p *provider) CleanExpiredMFASessions(ctx context.Context) error {
	currentTime := time.Now().Unix()
	f := expression.Name("expires_at").LessThan(expression.Value(currentTime))
	items, err := p.scanFilteredAll(ctx, schemas.Collections.MFASession, nil, &f)
	if err != nil {
		return err
	}
	for _, it := range items {
		var s schemas.MFASession
		if err := unmarshalItem(it, &s); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.MFASession, "id", s.ID); err != nil {
			return err
		}
	}
	return nil
}

// GetAllMFASessions retrieves all MFA sessions (for testing)
func (p *provider) GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error) {
	items, err := p.scanAllRaw(ctx, schemas.Collections.MFASession, nil, nil)
	if err != nil {
		return nil, err
	}
	var result []*schemas.MFASession
	for _, it := range items {
		var s schemas.MFASession
		if err := unmarshalItem(it, &s); err != nil {
			return nil, err
		}
		result = append(result, &s)
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
	existing, _ := p.queryEq(ctx, schemas.Collections.OAuthState, "state_key", "state_key", state.StateKey, nil)
	for _, it := range existing {
		var e schemas.OAuthState
		if err := unmarshalItem(it, &e); err != nil {
			continue
		}
		_ = p.deleteItemByHash(ctx, schemas.Collections.OAuthState, "id", e.ID)
	}
	return p.putItem(ctx, schemas.Collections.OAuthState, state)
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	items, err := p.queryEqLimit(ctx, schemas.Collections.OAuthState, "state_key", "state_key", key, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("OAuth state not found")
	}
	var s schemas.OAuthState
	if err := unmarshalItem(items[0], &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteOAuthStateByKey deletes an OAuth state by key
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	items, err := p.queryEq(ctx, schemas.Collections.OAuthState, "state_key", "state_key", key, nil)
	if err != nil {
		return err
	}
	for _, it := range items {
		var s schemas.OAuthState
		if err := unmarshalItem(it, &s); err != nil {
			return err
		}
		if err := p.deleteItemByHash(ctx, schemas.Collections.OAuthState, "id", s.ID); err != nil {
			return err
		}
	}
	return nil
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	items, err := p.scanAllRaw(ctx, schemas.Collections.OAuthState, nil, nil)
	if err != nil {
		return nil, err
	}
	var result []*schemas.OAuthState
	for _, it := range items {
		var s schemas.OAuthState
		if err := unmarshalItem(it, &s); err != nil {
			return nil, err
		}
		result = append(result, &s)
	}
	return result, nil
}
