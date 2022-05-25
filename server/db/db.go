package db

import (
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/providers"
	"github.com/authorizerdev/authorizer/server/db/providers/arangodb"
	"github.com/authorizerdev/authorizer/server/db/providers/cassandradb"
	"github.com/authorizerdev/authorizer/server/db/providers/mongodb"
	"github.com/authorizerdev/authorizer/server/db/providers/sql"
	"github.com/authorizerdev/authorizer/server/envstore"
)

// Provider returns the current database provider
var Provider providers.Provider

func InitDB() error {
	var err error

	isSQL := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeArangodb && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeMongodb && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeCassandraDB
	isArangoDB := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeArangodb
	isMongoDB := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeMongodb
	isCassandra := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeCassandraDB

	if isSQL {
		log.Info("Initializing SQL Driver for: ", envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType))
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

	return nil
}
