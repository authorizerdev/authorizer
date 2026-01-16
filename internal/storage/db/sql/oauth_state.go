package sql

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOAuthState adds an OAuth state to the database
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
	// Delete existing state with the same key first (upsert behavior)
	err := p.db.Where("state_key = ?", state.StateKey).Delete(&schemas.OAuthState{}).Error
	if err != nil {
		return err
	}
	return p.db.Create(state).Error
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	var state schemas.OAuthState
	err := p.db.Where("state_key = ?", key).First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// DeleteOAuthStateByKey deletes an OAuth state by key
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	return p.db.Where("state_key = ?", key).Delete(&schemas.OAuthState{}).Error
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	var states []*schemas.OAuthState
	err := p.db.Find(&states).Error
	return states, err
}
