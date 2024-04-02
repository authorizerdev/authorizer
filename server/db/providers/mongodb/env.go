package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env *models.Env) (*models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	env.Key = env.ID
	configCollection := p.db.Collection(models.Collections.Env, options.Collection())
	_, err := configCollection.InsertOne(ctx, env)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env *models.Env) (*models.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	configCollection := p.db.Collection(models.Collections.Env, options.Collection())
	_, err := configCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": env.ID}}, bson.M{"$set": env}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (*models.Env, error) {
	var env *models.Env
	configCollection := p.db.Collection(models.Collections.Env, options.Collection())
	cursor, err := configCollection.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(nil) {
		err := cursor.Decode(&env)
		if err != nil {
			return nil, err
		}
	}
	if env == nil {
		return env, fmt.Errorf("config not found")
	}
	return env, nil
}
