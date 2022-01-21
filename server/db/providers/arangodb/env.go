package arangodb

import (
	"fmt"
	"log"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/server/db/models"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(env models.Env) (models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
	configCollection, _ := p.db.Collection(nil, models.Collections.Env)
	meta, err := configCollection.CreateDocument(arangoDriver.WithOverwrite(nil), env)
	if err != nil {
		log.Println("error adding config:", err)
		return env, err
	}
	env.Key = meta.Key
	env.ID = meta.ID.String()
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(env models.Env) (models.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	collection, _ := p.db.Collection(nil, models.Collections.Env)
	meta, err := collection.UpdateDocument(nil, env.Key, env)
	if err != nil {
		log.Println("error updating config:", err)
		return env, err
	}

	env.Key = meta.Key
	env.ID = meta.ID.String()
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv() (models.Env, error) {
	var env models.Env
	query := fmt.Sprintf("FOR d in %s RETURN d", models.Collections.Env)

	cursor, err := p.db.Query(nil, query, nil)
	if err != nil {
		return env, err
	}
	defer cursor.Close()

	for {
		if !cursor.HasMore() {
			if env.Key == "" {
				return env, fmt.Errorf("config not found")
			}
			break
		}
		_, err := cursor.ReadDocument(nil, &env)
		if err != nil {
			return env, err
		}
	}

	return env, nil
}
