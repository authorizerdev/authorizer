package surrealdb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddOAuthState saves an OAuth state, replacing any state with the same key.
// The delete and the insert run in one transaction.
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
	return p.transaction(ctx, `DELETE type::table($tb) WHERE state_key = $k RETURN NONE;
		CREATE type::record($tb, $id) CONTENT $doc RETURN NONE;`,
		map[string]any{"tb": schemas.Collections.OAuthState, "k": state.StateKey, "id": state.ID, "doc": doc(state)})
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	return getWhere[schemas.OAuthState](ctx, p, schemas.Collections.OAuthState, "state_key = $v", "oauth state", map[string]any{"v": key})
}

// DeleteOAuthStateByKey deletes an OAuth state by key and reports whether this
// call deleted it, so exactly one concurrent caller can consume a state.
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) (bool, error) {
	return p.deleteReportingWinner(ctx, schemas.Collections.OAuthState, "state_key = $k", map[string]any{"k": key})
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	rows, err := listWhere[schemas.OAuthState](ctx, p, schemas.Collections.OAuthState, "", nil)
	return ptrs(rows), err
}
