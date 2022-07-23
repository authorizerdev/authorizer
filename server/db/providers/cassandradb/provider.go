package cassandradb

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/gocql/gocql"
	cansandraDriver "github.com/gocql/gocql"
)

type provider struct {
	db *cansandraDriver.Session
}

// KeySpace for the cassandra database
var KeySpace string

// NewProvider to initialize arangodb connection
func NewProvider() (*provider, error) {
	dbURL := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseURL
	if dbURL == "" {
		dbHost := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseHost
		dbPort := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabasePort
		if dbPort != "" && dbHost != "" {
			dbURL = fmt.Sprintf("%s:%s", dbHost, dbPort)
		} else if dbHost != "" {
			dbURL = dbHost
		}
	}

	KeySpace = memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseName
	if KeySpace == "" {
		KeySpace = constants.EnvKeyDatabaseName
	}
	clusterURL := []string{}
	if strings.Contains(dbURL, ",") {
		clusterURL = strings.Split(dbURL, ",")
	} else {
		clusterURL = append(clusterURL, dbURL)
	}
	cassandraClient := cansandraDriver.NewCluster(clusterURL...)
	dbUsername := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseUsername
	dbPassword := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabasePassword

	if dbUsername != "" && dbPassword != "" {
		cassandraClient.Authenticator = &cansandraDriver.PasswordAuthenticator{
			Username: dbUsername,
			Password: dbPassword,
		}
	}

	dbCert := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseCert
	dbCACert := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseCACert
	dbCertKey := memorystore.RequiredEnvStoreObj.GetRequiredEnv().DatabaseCertKey
	if dbCert != "" && dbCACert != "" && dbCertKey != "" {
		certString, err := crypto.DecryptB64(dbCert)
		if err != nil {
			return nil, err
		}

		keyString, err := crypto.DecryptB64(dbCertKey)
		if err != nil {
			return nil, err
		}

		caString, err := crypto.DecryptB64(dbCACert)
		if err != nil {
			return nil, err
		}

		cert, err := tls.X509KeyPair([]byte(certString), []byte(keyString))
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caString))

		cassandraClient.SslOpts = &cansandraDriver.SslOptions{
			Config: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				RootCAs:            caCertPool,
				InsecureSkipVerify: true,
			},
			EnableHostVerification: false,
		}
	}

	cassandraClient.RetryPolicy = &cansandraDriver.SimpleRetryPolicy{
		NumRetries: 3,
	}
	cassandraClient.Consistency = gocql.LocalQuorum
	cassandraClient.ConnectTimeout = 10 * time.Second
	cassandraClient.ProtoVersion = 4

	session, err := cassandraClient.CreateSession()
	if err != nil {
		return nil, err
	}

	// Note for astra keyspaces can only be created from there console
	// https://docs.datastax.com/en/astra/docs/datastax-astra-faq.html#_i_am_trying_to_create_a_keyspace_in_the_cql_shell_and_i_am_running_into_an_error_how_do_i_fix_this
	getKeyspaceQuery := fmt.Sprintf("SELECT keyspace_name FROM system_schema.keyspaces;")
	scanner := session.Query(getKeyspaceQuery).Iter().Scanner()
	hasAuthorizerKeySpace := false
	for scanner.Next() {
		var keySpace string
		err := scanner.Scan(&keySpace)
		if err != nil {
			return nil, err
		}
		if keySpace == KeySpace {
			hasAuthorizerKeySpace = true
			break
		}
	}

	if !hasAuthorizerKeySpace {
		createKeySpaceQuery := fmt.Sprintf("CREATE KEYSPACE %s WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1};", KeySpace)
		err = session.Query(createKeySpaceQuery).Exec()
		if err != nil {
			return nil, err
		}
	}

	// make sure collections are present
	envCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, env text, hash text, updated_at bigint, created_at bigint, PRIMARY KEY (id))",
		KeySpace, models.Collections.Env)
	err = session.Query(envCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}

	sessionCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, user_id text, user_agent text, ip text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, models.Collections.Session)
	err = session.Query(sessionCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	sessionIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_session_user_id ON %s.%s (user_id)", KeySpace, models.Collections.Session)
	err = session.Query(sessionIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	userCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, email text, email_verified_at bigint, password text, signup_methods text, given_name text, family_name text, middle_name text, nickname text, gender text, birthdate text, phone_number text, phone_number_verified_at bigint, picture text, roles text, updated_at bigint, created_at bigint, revoked_timestamp bigint, PRIMARY KEY (id))", KeySpace, models.Collections.User)
	err = session.Query(userCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	userIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_user_email ON %s.%s (email)", KeySpace, models.Collections.User)
	err = session.Query(userIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	// add is_multi_factor_auth_enabled on users table
	userTableAlterQuery := fmt.Sprintf(`ALTER TABLE %s.%s ADD is_multi_factor_auth_enabled boolean;`, KeySpace, models.Collections.User)
	err = session.Query(userTableAlterQuery).Exec()
	if err != nil {
		return nil, err
	}

	// token is reserved keyword in cassandra, hence we need to use jwt_token
	verificationRequestCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, jwt_token text, identifier text, expires_at bigint, email text, nonce text, redirect_uri text, created_at bigint, updated_at bigint, PRIMARY KEY (id))", KeySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	verificationRequestIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_email ON %s.%s (email)", KeySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	verificationRequestIndexQuery = fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_identifier ON %s.%s (identifier)", KeySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		return nil, err
	}
	verificationRequestIndexQuery = fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_verification_request_jwt_token ON %s.%s (jwt_token)", KeySpace, models.Collections.VerificationRequest)
	err = session.Query(verificationRequestIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	webhookCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, event_name text, endpoint text, enabled boolean, headers text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, models.Collections.Webhook)
	err = session.Query(webhookCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	webhookIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_webhook_event_name ON %s.%s (event_name)", KeySpace, models.Collections.Webhook)
	err = session.Query(webhookIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	webhookLogCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, http_status bigint, response text, request text, webhook_id text,updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, models.Collections.WebhookLog)
	err = session.Query(webhookLogCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	webhookLogIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_webhook_log_webhook_id ON %s.%s (webhook_id)", KeySpace, models.Collections.WebhookLog)
	err = session.Query(webhookLogIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	emailTemplateCollectionQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (id text, event_name text, template text, updated_at bigint, created_at bigint, PRIMARY KEY (id))", KeySpace, models.Collections.EmailTemplate)
	err = session.Query(emailTemplateCollectionQuery).Exec()
	if err != nil {
		return nil, err
	}
	emailTemplateIndexQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS authorizer_email_template_event_name ON %s.%s (event_name)", KeySpace, models.Collections.EmailTemplate)
	err = session.Query(emailTemplateIndexQuery).Exec()
	if err != nil {
		return nil, err
	}

	return &provider{
		db: session,
	}, err
}
