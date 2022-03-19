package cassandra

import (
	"fmt"
	"log"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	cansandraDriver "github.com/gocql/gocql"
)

type provider struct {
	db *cansandraDriver.Session
}

func (s provider) createTableIfNotExists(tableName string, fields []string) error {
	keySpace := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName)
	return s.db.Query(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (%s)`, keySpace+"."+tableName, strings.Join(fields, ", "))).Exec()
}

// NewProvider to initialize arangodb connection
func NewProvider() (*provider, error) {
	dbURL := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)
	keySpace := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName)
	cassandraClient := cansandraDriver.NewCluster(dbURL)
	cassandraClient.Keyspace = keySpace
	cassandraClient.RetryPolicy = &cansandraDriver.SimpleRetryPolicy{
		NumRetries: 3,
	}
	cassandraClient.Consistency = cansandraDriver.Quorum

	session, err := cassandraClient.CreateSession()
	if err != nil {
		log.Println("Error while creating connection to cassandra db", err)
		return nil, err
	}

	q := fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor':1}",
		keySpace)
	err = session.Query(q).Exec()
	if err != nil {
		log.Println("Unable to create keyspace:", err)
		return nil, err
	}

	// make sure collections are present

	return &provider{
		db: session,
	}, err
}
