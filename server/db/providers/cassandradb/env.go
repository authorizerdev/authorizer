package cassandradb

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env *models.Env) (*models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	insertEnvQuery := fmt.Sprintf("INSERT INTO %s (id, env, hash, created_at, updated_at) VALUES ('%s', '%s', '%s', %d, %d)", KeySpace+"."+models.Collections.Env, env.ID, env.EnvData, env.Hash, env.CreatedAt, env.UpdatedAt)
	err := p.db.Query(insertEnvQuery).Exec()
	if err != nil {
		return env, err
	}

	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env *models.Env) (*models.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	updateEnvQuery := fmt.Sprintf("UPDATE %s SET env = '%s', updated_at = %d WHERE id = '%s'", KeySpace+"."+models.Collections.Env, env.EnvData, env.UpdatedAt, env.ID)
	err := p.db.Query(updateEnvQuery).Exec()
	if err != nil {
		return env, err
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (*models.Env, error) {
	var env models.Env
	query := fmt.Sprintf("SELECT id, env, hash, created_at, updated_at FROM %s LIMIT 1", KeySpace+"."+models.Collections.Env)
	err := p.db.Query(query).Consistency(gocql.One).Scan(&env.ID, &env.EnvData, &env.Hash, &env.CreatedAt, &env.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &env, nil
}
