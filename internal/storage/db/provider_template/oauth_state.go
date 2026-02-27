package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// OAuth state methods implement the database-backed memory store for OAuth login state.
// Used when Redis is not configured; the memory_store/db provider delegates to these.
// Table/collection: schemas.Collections.OAuthState ("authorizer_oauth_states")
// Key field: state_key (maps to "key" param in Get/Delete)

// AddOAuthState adds an OAuth state to the database.
// Implement as upsert: delete existing state with same state_key, then insert.
// State fields: ID, StateKey, State, CreatedAt, UpdatedAt
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
	// TODO: upsert - delete where state_key = state.StateKey, then insert
	return nil
}

// GetOAuthStateByKey retrieves an OAuth state by key (StateKey).
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	// TODO: query where state_key = ?
	var state *schemas.OAuthState
	return state, nil
}

// DeleteOAuthStateByKey deletes an OAuth state by key (StateKey).
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	// TODO: delete where state_key = ?
	return nil
}

// GetAllOAuthStates retrieves all OAuth states (for testing).
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	// TODO: select all from schemas.Collections.OAuthState
	var states []*schemas.OAuthState
	return states, nil
}
