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

	isSQL := envs.DatabaseType != constants.DbTypeArangodb && envs.DatabaseType != constants.DbTypeMongodb && envs.DatabaseType != constants.DbTypeCassandraDB && envs.DatabaseType != constants.DbTypeScyllaDB && envs.DatabaseType != constants.DbTypeDynamoDB && envs.DatabaseType != constants.DbTypeCouchbaseDB
	isArangoDB := envs.DatabaseType == constants.DbTypeArangodb
	isMongoDB := envs.DatabaseType == constants.DbTypeMongodb
	isCassandra := envs.DatabaseType == constants.DbTypeCassandraDB || envs.DatabaseType == constants.DbTypeScyllaDB
	isDynamoDB := envs.DatabaseType == constants.DbTypeDynamoDB
	isCouchbaseDB := envs.DatabaseType == constants.DbTypeCouchbaseDB

	if isSQL {
		log.Info("Initializing SQL Driver for: ", envs.DatabaseType)
		Provider, err = sql.NewProvider()
		if err != nil {
			log.Fatal("Failed to initialize SQL driver: ", err)
			return err
		}
	}

	if isArangoDB {
		log.Info("Initializing ArangoDB Driver")
		Provider, err = arangodb.NewProvider()
		if err != nil {
			log.Fatal("Failed to initialize ArangoDB driver: ", err)
			return err
		}
	}

	if isMongoDB {
		log.Info("Initializing MongoDB Driver")
		Provider, err = mongodb.NewProvider()
		if err != nil {
			log.Fatal("Failed to initialize MongoDB driver: ", err)
			return err
		}
	}

	if isCassandra {
		log.Info("Initializing CassandraDB Driver")
		Provider, err = cassandradb.NewProvider()
		if err != nil {
			log.Fatal("Failed to initialize CassandraDB driver: ", err)
			return err
		}
	}

	if isDynamoDB {
		log.Info("Initializing DynamoDB Driver for: ", envs.DatabaseType)
		Provider, err = dynamodb.NewProvider()
		if err != nil {
			log.Fatal("Failed to initialize DynamoDB driver: ", err)
			return err
		}
	}

	if isCouchbaseDB {
		log.Info("Initializing CouchbaseDB Driver for: ", envs.DatabaseType)
		Provider, err = couchbase.NewProvider()
		if err != nil {
			log.Fatal("Failed to initialize Couchbase driver: ", err)
			return err
		}
	}

	return nil
}
