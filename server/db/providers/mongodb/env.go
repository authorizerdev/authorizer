package mongodb

import (
	"fmt"
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(env models.Env) (models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	env.Key = env.ID
	configCollection := p.db.Collection(models.Collections.Env, options.Collection())
	_, err := configCollection.InsertOne(nil, env)
	if err != nil {
		log.Println("error adding config:", err)
		return env, err
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(env models.Env) (models.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	configCollection := p.db.Collection(models.Collections.Env, options.Collection())
	_, err := configCollection.UpdateOne(nil, bson.M{"_id": bson.M{"$eq": env.ID}}, bson.M{"$set": env}, options.MergeUpdateOptions())
	if err != nil {
		log.Println("error updating config:", err)
		return env, err
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv() (models.Env, error) {
	var env models.Env
	configCollection := p.db.Collection(models.Collections.Env, options.Collection())
	cursor, err := configCollection.Find(nil, bson.M{}, options.Find())
	if err != nil {
		return env, err
	}
	defer cursor.Close(nil)

	for cursor.Next(nil) {
		err := cursor.Decode(&env)
		if err != nil {
			return env, err
		}
	}

	if env.ID == "" {
		return env, fmt.Errorf("config not found")
	}

	return env, nil
}
