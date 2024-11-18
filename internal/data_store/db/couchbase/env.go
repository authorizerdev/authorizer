package couchbase

import (
	"context"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/data_store/schemas"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	env.Key = env.ID
	env.EncryptionKey = env.Hash
	insertOpt := gocb.InsertOptions{
		Context: ctx,
	}
	_, err := p.db.Collection(schemas.Collections.Env).Insert(env.ID, env, &insertOpt)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	env.EncryptionKey = env.Hash

	updateEnvQuery := fmt.Sprintf("UPDATE %s.%s SET env = $1, updated_at = $2 WHERE _id = $3", p.scopeName, schemas.Collections.Env)
	_, err := p.db.Query(updateEnvQuery, &gocb.QueryOptions{
		Context:              ctx,
		PositionalParameters: []interface{}{env.EnvData, env.UpdatedAt, env.UpdatedAt, env.ID},
	})
	if err != nil {
		return nil, err
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (*schemas.Env, error) {
	var env *schemas.Env

	query := fmt.Sprintf("SELECT _id, env, encryption_key, created_at, updated_at FROM %s.%s LIMIT 1", p.scopeName, schemas.Collections.Env)
	q, err := p.db.Query(query, &gocb.QueryOptions{
		Context:         ctx,
		ScanConsistency: gocb.QueryScanConsistencyRequestPlus,
	})
	if err != nil {
		return nil, err
	}
	err = q.One(&env)
	if err != nil {
		return nil, err
	}
	env.Hash = env.EncryptionKey
	return env, nil
}
