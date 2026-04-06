package db

import (
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
)

// storageDBEntry matches one entry for memory store DB tests.
// Memory store DB tests only run against SQLite — storage-layer compatibility
// is covered by internal/storage tests.
type storageDBEntry struct {
	dbType string
	dbURL  string
}

func storageTestDBEntriesFromEnv() []storageDBEntry {
	return []storageDBEntry{
		{dbType: constants.DbTypeSqlite, dbURL: "test.db"},
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

	return cfg
}
