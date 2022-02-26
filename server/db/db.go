package db

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/providers"
	"github.com/authorizerdev/authorizer/server/db/providers/arangodb"
	"github.com/authorizerdev/authorizer/server/db/providers/mongodb"
	"github.com/authorizerdev/authorizer/server/db/providers/sql"
	"github.com/authorizerdev/authorizer/server/envstore"
)

// Provider returns the current database provider
var Provider providers.Provider

func InitDB() error {
	var err error

	isSQL := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeArangodb && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) != constants.DbTypeMongodb
	isArangoDB := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeArangodb
	isMongoDB := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseType) == constants.DbTypeMongodb

	if isSQL {
		Provider, err = sql.NewProvider()
		if err != nil {
			return err
		}
	}

	if isArangoDB {
		Provider, err = arangodb.NewProvider()
		if err != nil {
			return err
		}
	}

	if isMongoDB {
		Provider, err = mongodb.NewProvider()
		if err != nil {
			return err
		}
	}

	return nil
}
