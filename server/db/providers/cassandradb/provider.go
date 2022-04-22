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

// KeySpace for the cassandra database
var KeySpace string

// NewProvider to initialize arangodb connection
func NewProvider() (*provider, error) {
	dbURL := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseURL)
	KeySpace = envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyDatabaseName)
	cassandraClient := cansandraDriver.NewCluster(dbURL)

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
		KeySpace)
	err = session.Query(keyspaceQuery).Exec()
	if err != nil {
		log.Println("Unable to create keyspace:", err)
		return nil, err
	}

	// make sure collections are present
	envCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, env text, hash text, updated_at bigint, created_at bigint, PRIMARY KEY (id))",
		KeySpace, models.Collections.Env)
	err = session.Query(envCollectionQuery).Exec()
	if err != nil {
		log.Println("Unable to create env collection:", err)
		return nil, err
	}

	sessionCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, user_id text, user_agent text, ip text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, models.Collections.Session)
	err = session.Query(sessionCollectionQuery).Exec()
	if err != nil {
		log.Println("Unable to create session collection:", err)
		return nil, err
	}

	userCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, email text, email_verified_at bigint, password text, signup_methods text, given_name text, family_name text, middle_name text, nickname text, gender text, birthdate text, phone_number text, phone_number_verified_at bigint, picture text, roles text, updated_at bigint, created_at bigint, revoked_timestamp bigint, PRIMARY KEY (id))", KeySpace, models.Collections.User)
	err = session.Query(userCollectionQuery).Exec()
	if err != nil {
		log.Println("Unable to create user collection:", err)
		return nil, err
	}
	userIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_user_email ON %s.%s (email)", KeySpace, models.Collections.User)
	err = session.Query(userIndexQuery).Exec()
	if err != nil {
		log.Println("Unable to create user index:", err)
		return nil, err
	}

	// token is reserved keyword in cassandra, hence we need to use jwt_token
	verificationRequestCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, jwt_token text, identifier text, expires_at bigint, email text, nonce text, redirect_uri text, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestCollectionQuery).Exec()
	if err != nil {
		log.Println("Unable to create verification request collection:", err)
		return nil, err
	}
	verificationRequestIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_email ON %s.%s (email)", KeySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		log.Println("Unable to create verification_requests index:", err)
		return nil, err
	}
	verificationRequestIndexQuery = fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_identifier ON %s.%s (identifier)", KeySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		log.Println("Unable to create verification_requests index:", err)
		return nil, err
	}
	verificationRequestIndexQuery = fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_jwt_token ON %s.%s (jwt_token)", KeySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		log.Println("Unable to create verification_requests index:", err)
		return nil, err
	}

	return &provider{
		db: session,
	}, err
}
