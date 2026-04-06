package integration_tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/graphql"
	"github.com/authorizerdev/authorizer/internal/http_handlers"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// testSetup represents the test setup
type testSetup struct {
	GraphQLProvider graphql.Provider
	HttpProvider    http_handlers.Provider
	HttpServer      *httptest.Server
	Config          *config.Config
	Logger          *zerolog.Logger
	GinContext      *gin.Context
	// Used for specific tests where we need to access the storage
	StorageProvider     storage.Provider
	MemoryStoreProvider memory_store.Provider
}

func createContext(s *testSetup) (*http.Request, context.Context) {
	req, err := http.NewRequest(
		http.MethodPost,
		"http://"+s.HttpServer.Listener.Addr().String()+"/graphql",
		nil,
	)
	if err != nil {
		panic("integration_tests.createContext: " + err.Error())
	}

	ctx := utils.ContextWithGin(req.Context(), s.GinContext)
	s.GinContext.Request = req
	return req, ctx
}

// dbTestConfig holds database-specific test configuration
type dbTestConfig struct {
	DbType string
	DbURL  string
}

// getTestDBs returns the list of database configurations to test against.
// It reads the TEST_DBS environment variable (comma-separated list of db types).
// Defaults to "postgres" if not set.
func getTestDBs() []dbTestConfig {
	testDBsEnv := os.Getenv("TEST_DBS")
	if testDBsEnv == "" {
		testDBsEnv = "postgres"
	}

	dbTypes := strings.Split(testDBsEnv, ",")
	var configs []dbTestConfig

	for _, dbType := range dbTypes {
		dbType = strings.TrimSpace(dbType)
		if dbType == "" {
			continue
		}

		dbURL := getDBURL(dbType)
		if dbURL != "" {
			configs = append(configs, dbTestConfig{
				DbType: dbType,
				DbURL:  dbURL,
			})
		}
	}

	return configs
}

// getDBURL returns the connection URL for a given database type
func getDBURL(dbType string) string {
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
		return "http://localhost:8000"
	case constants.DbTypeCouchbaseDB:
		return "couchbase://localhost"
	default:
		return ""
	}
}

// getTestConfig returns a test config for the default database (postgres).
// For multi-DB testing, use runForEachDB instead.
func getTestConfig() *config.Config {
	return getTestConfigForDB(constants.DbTypePostgres, "postgres://postgres:postgres@localhost:5434/postgres")
}

// getTestConfigForDB returns a test config for a specific database type and URL
func getTestConfigForDB(dbType, dbURL string) *config.Config {
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

	// Set MongoDB-specific config
	if dbType == constants.DbTypeMongoDB {
		cfg.DatabaseName = "authorizer_test"
	}

	// Set Couchbase-specific config
	if dbType == constants.DbTypeCouchbaseDB {
		cfg.DatabaseUsername = "Administrator"
		cfg.DatabasePassword = "password"
		cfg.CouchBaseBucket = "authorizer_test"
	}

	return cfg
}

// runForEachDB runs the given test function against each database specified in TEST_DBS.
// This is the primary way to run tests across multiple database providers.
//
// Usage:
//
//	func TestFeature(t *testing.T) {
//	    runForEachDB(t, func(t *testing.T, cfg *config.Config) {
//	        ts := initTestSetup(t, cfg)
//	        _, ctx := createContext(ts)
//	        // ... test logic
//	    })
//	}
func runForEachDB(t *testing.T, testFn func(t *testing.T, cfg *config.Config)) {
	t.Helper()
	dbConfigs := getTestDBs()
	if len(dbConfigs) == 0 {
		t.Fatal("TEST_DBS produced no runnable database configurations; check TEST_DBS and that each database type resolves to a non-empty URL")
	}

	for _, dbCfg := range dbConfigs {
		t.Run("db="+dbCfg.DbType, func(t *testing.T) {
			cfg := getTestConfigForDB(dbCfg.DbType, dbCfg.DbURL)
			testFn(t, cfg)
		})
	}
}

// initTestSetup initializes the test setup
func initTestSetup(t *testing.T, cfg *config.Config) *testSetup {
	// Initialize logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	if cfg.DatabaseType == constants.DbTypeSqlite || cfg.DatabaseType == constants.DbTypeLibSQL {
		cfg.DatabaseURL = filepath.Join(t.TempDir(), "authorizer_integration.db")
	}

	// Initialize storage provider first as it's required by other providers
	storageProvider, err := storage.New(cfg, &storage.Dependencies{
		Log: &logger,
	})
	require.NoError(t, err)

	// Initialize other providers
	authProvider, err := authenticators.New(cfg, &authenticators.Dependencies{
		Log:             &logger,
		StorageProvider: storageProvider,
	})
	require.NoError(t, err)

	emailProvider, err := email.New(cfg, &email.Dependencies{
		Log:             &logger,
		StorageProvider: storageProvider,
	})
	require.NoError(t, err)

	eventsProvider, err := events.New(cfg, &events.Dependencies{
		Log:             &logger,
		StorageProvider: storageProvider,
	})
	require.NoError(t, err)

	memoryStoreProvider, err := memory_store.New(cfg, &memory_store.Dependencies{
		Log: &logger,
	})
	require.NoError(t, err)

	smsProvider, err := sms.New(cfg, &sms.Dependencies{
		Log: &logger,
	})
	require.NoError(t, err)

	tokenProvider, err := token.New(cfg, &token.Dependencies{
		Log:                 &logger,
		MemoryStoreProvider: memoryStoreProvider,
	})
	require.NoError(t, err)

	rateLimitProvider, err := rate_limit.New(cfg, &rate_limit.Dependencies{
		Log: &logger,
	})
	require.NoError(t, err)

	oauthProvider, err := oauth.New(cfg, &oauth.Dependencies{
		Log: &logger,
	})
	require.NoError(t, err)

	// Initialize audit provider
	auditProvider := audit.New(&audit.Dependencies{
		Log:             &logger,
		StorageProvider: storageProvider,
	})

	// Create dependencies struct
	gqlDeps := &graphql.Dependencies{
		Log:                   &logger,
		AuditProvider:         auditProvider,
		AuthenticatorProvider: authProvider,
		EmailProvider:         emailProvider,
		EventsProvider:        eventsProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		SMSProvider:           smsProvider,
		StorageProvider:       storageProvider,
		TokenProvider:         tokenProvider,
	}

	// Create dependencies struct
	httpDeps := &http_handlers.Dependencies{
		Log:                   &logger,
		AuditProvider:         auditProvider,
		AuthenticatorProvider: authProvider,
		EmailProvider:         emailProvider,
		EventsProvider:        eventsProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		SMSProvider:           smsProvider,
		StorageProvider:       storageProvider,
		TokenProvider:         tokenProvider,
		RateLimitProvider:     rateLimitProvider,
		OAuthProvider:         oauthProvider,
	}

	// Create GraphQL provider
	gqlProvider, err := graphql.New(cfg, gqlDeps)
	require.NoError(t, err)

	// Create HTTP provider
	httpProvider, err := http_handlers.New(cfg, httpDeps)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	ctx, r := gin.CreateTestContext(w)
	r.Use(httpProvider.CORSMiddleware())
	r.Use(httpProvider.ContextMiddleware())
	r.Use(httpProvider.LoggerMiddleware())

	r.POST("/graphql", httpProvider.GraphqlHandler())

	server := httptest.NewServer(r)

	t.Cleanup(func() {
		server.Close()
		if storageProvider != nil {
			if err := storageProvider.Close(); err != nil {
				t.Errorf("close storage provider: %v", err)
			}
		}
	})

	return &testSetup{
		GraphQLProvider:     gqlProvider,
		HttpProvider:        httpProvider,
		HttpServer:          server,
		Logger:              &logger,
		GinContext:          ctx,
		StorageProvider:     storageProvider,
		MemoryStoreProvider: memoryStoreProvider,
	}
}
