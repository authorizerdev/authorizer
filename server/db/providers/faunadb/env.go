package faunadb

import (
	"log"
	"time"

	f "github.com/fauna/faunadb-go/v5/faunadb"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/server/db/models"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(env models.Env) (models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
		env.Key = env.ID
	}

	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()

	_, err := p.db.Query(
		f.Create(
			f.Collection(models.Collections.Env),
			f.Obj{
				"data": env,
			},
		),
	)
	if err != nil {
		log.Println("error adding env:", err)
		return env, err
	}

	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(env models.Env) (models.Env, error) {
	env.UpdatedAt = time.Now().Unix()

	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv() (models.Env, error) {
	var env models.Env

	return env, nil
}
