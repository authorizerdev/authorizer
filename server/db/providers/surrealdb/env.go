package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"github.com/surrealdb/surrealdb.go"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env models.Env) (models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()

	mapData := env.ToMap()
	mapData[models.SurrealDbIdentifier] = env.ID

	_, err := p.db.Create(models.Collections.Env, mapData)
	if err != nil {
		return env, err
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env models.Env) (models.Env, error) {
	env.UpdatedAt = time.Now().Unix()

	mapData := env.ToMap()
	mapData[models.SurrealDbIdentifier] = env.ID

	_, err := p.db.Update(models.Collections.Env, mapData)
	if err != nil {
		return env, err
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (models.Env, error) {
	var env models.Env

	mapData, err := p.db.Select(models.Collections.Env)
	if err != nil {
		return env, err
	}

	envs := []models.Env{}
	err = surrealdb.Unmarshal(mapData, &envs)
	if err != nil {
		return env, err
	}

	if len(envs) > 0 {
		env = envs[0]
	} else {
		return env, fmt.Errorf("env record not found")
	}

	return env, nil
}
