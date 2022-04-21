package cassandradb

import (
	"fmt"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	cansandraDriver "github.com/gocql/gocql"
)

type provider struct {
	db *cansandraDriver.Session
}

// NewProvider to initialize arangodb connection
func NewProvider() (*provider, error) {
	dbURL := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)
	keySpace := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName)
	cassandraClient := cansandraDriver.NewCluster(dbURL)
	// cassandraClient.Keyspace = keySpace
	cassandraClient.RetryPolicy = &cansandraDriver.SimpleRetryPolicy{
		NumRetries: 3,
	}
	cassandraClient.Consistency = cansandraDriver.Quorum

	session, err := cassandraClient.CreateSession()
	if err != nil {
		log.Println("Error while creating connection to cassandra db", err)
		return nil, err
	}

	keyspaceQuery := fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor':1}",
		keySpace)
	err = session.Query(keyspaceQuery).Exec()
	if err != nil {
		log.Println("Unable to create keyspace:", err)
		return nil, err
	}

	// make sure collections are present
	envCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, env text, hash text, updated_at bigint, created_at bigint, PRIMARY KEY (id))",
		keySpace, models.Collections.Env)
	err = session.Query(envCollectionQuery).Exec()
	if err != nil {
		log.Println("Unable to create env collection:", err)
		return nil, err
	}

	sessionCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, user_id text, user_agent text, ip text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", keySpace, models.Collections.Session)
	err = session.Query(sessionCollectionQuery).Exec()
	if err != nil {
		log.Println("Unable to create session collection:", err)
		return nil, err
	}

	userCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, email text, email_verified_at bigint, password text, signup_methods text, given_name text, family_name text, middle_name text, nick_name text, gender text, birthdate text, phone_number text, phone_number_verified_at bigint, picture text, roles text, updated_at bigint, created_at bigint, revoked_timestamp bigint, PRIMARY KEY (id, email))", keySpace, models.Collections.User)
	err = session.Query(userCollectionQuery).Exec()
	if err != nil {
		log.Println("Unable to create user collection:", err)
		return nil, err
	}

	// token is reserved keyword in cassandra, hence we need to use jwt_token
	verificationRequestCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, jwt_token text, identifier text, expires_at bigint, email text, nonce text, redirect_uri text, created_at bigint, updated_at bigint, PRIMARY KEY (id, identifier, email))", keySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestCollectionQuery).Exec()
	if err != nil {
		log.Println("Unable to create verification request collection:", err)
		return nil, err
	}

	return &provider{
		db: session,
	}, err
}
