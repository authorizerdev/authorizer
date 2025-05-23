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

// AddEnv to save environment information in database
func (p *provider) AddEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	env.Key = env.ID
	configCollection := p.db.Collection(schemas.Collections.Env, options.Collection())
	_, err := configCollection.InsertOne(ctx, env)
	if err != nil {
		return nil, err
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	configCollection := p.db.Collection(schemas.Collections.Env, options.Collection())
	_, err := configCollection.UpdateOne(ctx, bson.M{"_id": bson.M{"$eq": env.ID}}, bson.M{"$set": env}, options.MergeUpdateOptions())
	if err != nil {
		return nil, err
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv(ctx context.Context) (*schemas.Env, error) {
	var env *schemas.Env
	configCollection := p.db.Collection(schemas.Collections.Env, options.Collection())
	cursor, err := configCollection.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
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
