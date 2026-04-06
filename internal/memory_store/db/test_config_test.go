package db

import (
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
)

// storageDBEntry matches one entry from TEST_DBS (same URLs as internal/integration_tests
// getTestDBs / getDBURL — keep these in sync when adding backends).
type storageDBEntry struct {
	dbType string
	dbURL  string
}

func storageTestDBEntriesFromEnv() []storageDBEntry {
	testDBsEnv := os.Getenv("TEST_DBS")
	if testDBsEnv == "" {
		testDBsEnv = "postgres"
	}
	var out []storageDBEntry
	for _, dbType := range strings.Split(testDBsEnv, ",") {
		dbType = strings.TrimSpace(dbType)
		if dbType == "" {
			continue
		}
		u := dbURLForMemoryStoreStorageTest(dbType)
		if u == "" {
			continue
		}
		out = append(out, storageDBEntry{dbType: dbType, dbURL: u})
	}
	return out
}

func dbURLForMemoryStoreStorageTest(dbType string) string {
	switch dbType {
	case constants.DbTypePostgres:
		return "postgres://postgres:postgres@localhost:5434/postgres"
	case constants.DbTypeSqlite:
		return "test.db"
	case constants.DbTypeLibSQL:
		return "test.db"
	case constants.DbTypeMysql:
		return "root:password@tcp(localhost:3306)/authorizer"
	case constants.DbTypeMariaDB:
		return "root:password@tcp(localhost:3307)/authorizer"
	case constants.DbTypeSqlserver:
		return "sqlserver://sa:Password123@localhost:1433?database=authorizer"
	case constants.DbTypeMongoDB:
		return "mongodb://localhost:27017"
	case constants.DbTypeArangoDB:
		return "http://localhost:8529"
	case constants.DbTypeScyllaDB, constants.DbTypeCassandraDB:
		return "127.0.0.1:9042"
	case constants.DbTypeDynamoDB:
		return "http://127.0.0.1:8000"
	case constants.DbTypeCouchbaseDB:
		return "couchbase://localhost"
	default:
		return ""
	}
}

func resolveSQLiteTestURL(dbType, mappedURL, tempPath string) string {
	if dbType == constants.DbTypeSqlite || dbType == constants.DbTypeLibSQL {
		return tempPath
	}
	return mappedURL
}

func buildStorageTestConfigForMemoryStore(dbType, dbURL string) *config.Config {
	cfg := &config.Config{
		Env:                             constants.TestEnv,
		SkipTestEndpointSSRFValidation:  true,
		DatabaseType:                    dbType,
		DatabaseURL:                     dbURL,
		JWTSecret:                       "test-secret",
		ClientID:                        "test-client-id",
		ClientSecret:                    "test-client-secret",
		AllowedOrigins:                  []string{"http://localhost:3000"},
		JWTType:                         "HS256",
		AdminSecret:                     "test-admin-secret",
		TwilioAPISecret:                 "test-twilio-api-secret",
		TwilioAPIKey:                    "test-twilio-api-key",
		TwilioAccountSID:                "test-twilio-account-sid",
		TwilioSender:                    "test-twilio-sender",
		DefaultRoles:                    []string{"user"},
		EnableSignup:                    true,
		EnableBasicAuthentication:       true,
		EnableMobileBasicAuthentication: true,
		EnableLoginPage:                 true,
		EnableStrongPassword:            true,
		IsSMSServiceEnabled:             true,
	}

	if dbType == constants.DbTypeMongoDB {
		cfg.DatabaseName = "authorizer_test"
	}

	if dbType == constants.DbTypeCouchbaseDB {
		cfg.DatabaseUsername = "Administrator"
		cfg.DatabasePassword = "password"
		cfg.CouchBaseBucket = "authorizer_test"
	}

	if dbType == constants.DbTypeDynamoDB {
		cfg.AWSRegion = "us-east-1"
	}

	return cfg
}
