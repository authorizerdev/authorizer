package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	collection := p.db.Table(schemas.Collections.Env)
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	env.Key = env.ID
	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	err := collection.Put(env).RunWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	collection := p.db.Table(schemas.Collections.Env)
	env.UpdatedAt = time.Now().Unix()
	err := UpdateByHashKey(collection, "id", env.ID, env)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (*schemas.Env, error) {
	var env *schemas.Env
	collection := p.db.Table(schemas.Collections.Env)
	// As there is no Find one supported.
	iter := collection.Scan().Limit(1).Iter()
	for iter.NextWithContext(ctx, &env) {
		if env == nil {
			return nil, errors.New("no documets found")
		} else {
			return env, nil
		}
	}
	err := iter.Err()
	if err != nil {
		return env, fmt.Errorf("config not found")
	}
	return env, nil
}
