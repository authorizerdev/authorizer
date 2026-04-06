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
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	env.Key = env.ID
	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	if err := p.putItem(ctx, schemas.Collections.Env, env); err != nil {
		return nil, err
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	if err := p.updateByHashKey(ctx, schemas.Collections.Env, "id", env.ID, env); err != nil {
		return nil, err
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (*schemas.Env, error) {
	items, err := p.scanFilteredLimit(ctx, schemas.Collections.Env, nil, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("no documets found")
	}
	var env schemas.Env
	if err := unmarshalItem(items[0], &env); err != nil {
		return nil, fmt.Errorf("config not found")
	}
	return &env, nil
}
