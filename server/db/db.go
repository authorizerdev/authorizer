package db

import (
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/providers"
	"github.com/authorizerdev/authorizer/server/db/providers/arangodb"
	"github.com/authorizerdev/authorizer/server/db/providers/cassandradb"
	"github.com/authorizerdev/authorizer/server/db/providers/couchbase"
	"github.com/authorizerdev/authorizer/server/db/providers/dynamodb"
	"github.com/authorizerdev/authorizer/server/db/providers/mongodb"
	"github.com/authorizerdev/authorizer/server/db/providers/sql"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

// Provider returns the current database provider
var Provider providers.Provider

func InitDB() error {
	var err error

	envs := memorystore.RequiredEnvStoreObj.GetRequiredEnv()

	switch envs.DatabaseType {
	case constants.DbTypeArangodb:
		log.Info("Initializing ArangoDB Driver")
		Provider, err = arangodb.NewProvider()

	case constants.DbTypeMongodb:
		log.Info("Initializing MongoDB Driver")
		Provider, err = mongodb.NewProvider()

	case constants.DbTypeCassandraDB, constants.DbTypeScyllaDB:
		log.Info("Initializing CassandraDB Driver")
		Provider, err = cassandradb.NewProvider()

	case constants.DbTypeDynamoDB:
		log.Info("Initializing DynamoDB Driver for: ", envs.DatabaseType)
		Provider, err = dynamodb.NewProvider()

	case constants.DbTypeCouchbaseDB:
		log.Info("Initializing CouchbaseDB Driver for: ", envs.DatabaseType)
		Provider, err = couchbase.NewProvider()

	default:
		log.Info("Initializing SQL Driver for: ", envs.DatabaseType)
		Provider, err = sql.NewProvider()
	}

	if err != nil {
		log.Fatal("Failed to initialize database driver: ", err)
		return err
	}

	return nil
}
