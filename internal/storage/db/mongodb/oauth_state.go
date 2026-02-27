package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

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
	collection := p.db.Collection(schemas.Collections.OAuthState, options.Collection())
	// Delete existing state with the same key first (upsert behavior)
	collection.DeleteMany(ctx, bson.M{"state_key": state.StateKey})
	_, err := collection.InsertOne(ctx, state)
	return err
}

// GetOAuthStateByKey retrieves an OAuth state by key
func (p *provider) GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error) {
	var state schemas.OAuthState
	collection := p.db.Collection(schemas.Collections.OAuthState, options.Collection())
	err := collection.FindOne(ctx, bson.M{"state_key": key}).Decode(&state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// DeleteOAuthStateByKey deletes an OAuth state by key
func (p *provider) DeleteOAuthStateByKey(ctx context.Context, key string) error {
	collection := p.db.Collection(schemas.Collections.OAuthState, options.Collection())
	_, err := collection.DeleteMany(ctx, bson.M{"state_key": key})
	return err
}

// GetAllOAuthStates retrieves all OAuth states (for testing)
func (p *provider) GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error) {
	var states []*schemas.OAuthState
	collection := p.db.Collection(schemas.Collections.OAuthState, options.Collection())
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &states)
	if err != nil {
		return nil, fmt.Errorf("failed to decode OAuth states: %w", err)
	}
	return states, nil
}
