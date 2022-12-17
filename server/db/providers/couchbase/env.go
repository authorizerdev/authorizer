package couchbase

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env models.Env) (models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	env.Key = env.ID

	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(models.Collections.Env).Insert(env.ID, env, &insertOpt)
	if err != nil {
		return env, err
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env models.Env) (models.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	scope := p.db.Scope("_default")

	updateEnvQuery := fmt.Sprintf("UPDATE auth._default.%s SET env = $1, updated_at = $2 WHERE _id = $3", models.Collections.Env)
	_, err := scope.Query(updateEnvQuery, &gocb.QueryOptions{
		Context:              ctx,
		PositionalParameters: []interface{}{env.EnvData, env.UpdatedAt, env.UpdatedAt, env.ID},
	})

	if err != nil {
		return env, err
	}

	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (models.Env, error) {
	var env models.Env
	scope := p.db.Scope("_default")
	query := fmt.Sprintf("SELECT _id, env, created_at, updated_at FROM auth._default.%s LIMIT 1", models.Collections.Env)
	q, err := scope.Query(query, &gocb.QueryOptions{
		Context: ctx,
	})
	if err != nil {
		return env, err
	}
	err = q.One(&env)

	if err != nil {
		return env, err
	}
	return env, nil
}
