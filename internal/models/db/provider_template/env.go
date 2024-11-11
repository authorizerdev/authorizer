package provider_template

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/models/schemas"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (*schemas.Env, error) {
	var env *schemas.Env

	return env, nil
}
