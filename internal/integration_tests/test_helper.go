package integration_tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	StorageProvider       storage.Provider
	MemoryStoreProvider   memory_store.Provider
	AuthenticatorProvider authenticators.Provider
	TokenProvider         token.Provider
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

// getTestConfig returns config for integration tests using SQLite.
// Integration tests validate business logic, not storage compatibility.
func getTestConfig() *config.Config {
	return getTestConfigForDB(constants.DbTypeSqlite, "test.db")
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

	// MongoDB, ArangoDB, Cassandra/Scylla require DatabaseName (keyspace / DB name); see storage New().
	if dbType == constants.DbTypeMongoDB || dbType == constants.DbTypeArangoDB ||
		dbType == constants.DbTypeScyllaDB || dbType == constants.DbTypeCassandraDB {
		cfg.DatabaseName = "authorizer_test"
	}

	// Set Couchbase-specific config
	if dbType == constants.DbTypeCouchbaseDB {
		cfg.DatabaseUsername = "Administrator"
		cfg.DatabasePassword = "password"
		cfg.CouchBaseBucket = "authorizer_test"
	}

	// DynamoDB Local (and AWS) expect a region for signing; avoid picking up real AWS keys in tests.
	if dbType == constants.DbTypeDynamoDB {
		cfg.AWSRegion = "us-east-1"
	}

	return cfg
}

// initTestSetup initializes the test setup
func initTestSetup(t *testing.T, cfg *config.Config) *testSetup {
	// Initialize logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	if cfg.DatabaseType == constants.DbTypeDynamoDB {
		// Match storage tests: use static creds from config instead of ambient AWS_* env.
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}

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
		GraphQLProvider:       gqlProvider,
		HttpProvider:          httpProvider,
		HttpServer:            server,
		Config:                cfg,
		Logger:                &logger,
		GinContext:            ctx,
		StorageProvider:       storageProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		AuthenticatorProvider: authProvider,
		TokenProvider:         tokenProvider,
	}
}
