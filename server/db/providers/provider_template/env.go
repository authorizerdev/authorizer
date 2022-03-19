package provider_template

import (
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/google/uuid"
)

// AddEnv to save environment information in database
func (p *provider) AddEnv(env models.Env) (models.Env, error) {
	if env.ID == "" {
		env.ID = uuid.New().String()
	}

	env.CreatedAt = time.Now().Unix()
	env.UpdatedAt = time.Now().Unix()
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
