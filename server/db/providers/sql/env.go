package sql

import (
	"log"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(env models.Env) (models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	env.Key = env.ID
	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()

	result := p.db.Create(&env)
	if result.Error != nil {
		log.Println("error adding config:", result.Error)
		return env, result.Error
	}
	return env, nil
}

// UpdateEnv to update environment information in database
func (p *provider) UpdateEnv(env models.Env) (models.Env, error) {
	env.UpdatedAt = time.Now().Unix()
	result := p.db.Save(&env)

	if result.Error != nil {
		log.Println("error updating config:", result.Error)
		return env, result.Error
	}
	return env, nil
}

// GetEnv to get environment information from database
func (p *provider) GetEnv() (models.Env, error) {
	var env models.Env
	result := p.db.First(&env)

	if result.Error != nil {
		return env, result.Error
	}

	return env, nil
}
