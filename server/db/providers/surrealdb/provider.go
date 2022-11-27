package surrealdb

import (
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/surrealdb/surrealdb.go"
)

// TODO change following provider to new db provider
type provider struct {
	db *surrealdb.DB
}

// NewProvider returns a new SQL provider
// TODO change following provider to new db provider
func NewProvider() (*provider, error) {
	dbURL := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseURL
	db, err := surrealdb.New(dbURL)
	if err != nil {
		return nil, err
	}

	dbUsername := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseUsername
	dbPassword := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabasePassword

	_, err = db.Signin(map[string]interface{}{
		"user": dbUsername,
		"pass": dbPassword,
	})
	if err != nil {
		return nil, err
	}

	_, err = db.Use(models.DBNamespace, models.DBNamespace)
	if err != nil {
		return nil, err
	}

	return &provider{
		db: db,
	}, nil
}
