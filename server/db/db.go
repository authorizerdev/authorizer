package db

import (
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/providers"
	"github.com/authorizerdev/authorizer/server/db/providers/arangodb"
	"github.com/authorizerdev/authorizer/server/db/providers/faunadb"
	"github.com/authorizerdev/authorizer/server/db/providers/mongodb"
	"github.com/authorizerdev/authorizer/server/db/providers/sql"
	"github.com/authorizerdev/authorizer/server/envstore"
)

// Provider returns the current database provider
var Provider providers.Provider

func InitDB() {
	var err error

	isSQL := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeArangodb && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeMongodb && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeFaunadb
	isArangoDB := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeArangodb
	isMongoDB := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeMongodb
	isFaunaDB := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeFaunadb

	if isSQL {
		Provider, err = sql.NewProvider()
		if err != nil {
			log.Fatal("=> error setting sql provider:", err)
		}
	}

	if isArangoDB {
		Provider, err = arangodb.NewProvider()
		if err != nil {
			log.Fatal("=> error setting arangodb provider:", err)
		}
	}

	if isMongoDB {
		Provider, err = mongodb.NewProvider()
		if err != nil {
			log.Fatal("=> error setting arangodb provider:", err)
		}
	}

	if isFaunaDB {
		Provider, err = faunadb.NewProvider()
		if err != nil {
			log.Fatal("=> error setting arangodb provider:", err)
		}
	}
}
