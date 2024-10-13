package arangodb

import (
	"context"
	"fmt"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/db/models"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env *models.Env) (*models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
		env.Key = env.ID
	}

	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	configCollection, _ := p.db.Collection(ctx, models.Collections.Env)
	meta, err := configCollection.CreateDocument(arangoDriver.WithOverwrite(ctx), env)
	if err != nil {
		return nil, err
	}
	env.Key = meta.Key
	env.ID = meta.ID.String()
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env *models.Env) (*models.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(ctx, models.Collections.Env)
	meta, err := collection.UpdateDocument(ctx, env.Key, env)
	if err != nil {
		return nil, err
	}

	env.Key = meta.Key
	env.ID = meta.ID.String()
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (*models.Env, error) {
	var env *models.Env
	query := fmt.Sprintf("FOR d in %s RETURN d", models.Collections.Env)
	cursor, err := p.db.Query(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	for {
		if !cursor.HasMore() {
			if env == nil {
				return env, fmt.Errorf("config not found")
			}
			break
		}
		_, err := cursor.ReadDocument(ctx, &env)
		if err != nil {
			return nil, err
		}
	}

	return env, nil
}
