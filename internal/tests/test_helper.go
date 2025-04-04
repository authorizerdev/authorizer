package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/graphql"
	"github.com/authorizerdev/authorizer/internal/http_handlers"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
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
	req, _ := http.NewRequest(
		"POST",
		"http://"+s.HttpServer.Listener.Addr().String()+"/graphql",
		nil,
	)

	ctx := context.WithValue(req.Context(), "GinContextKey", s.GinContext)
	s.GinContext.Request = req
	return req, ctx
}

func getTestConfig() *config.Config {
	// Initialize config with test settings
	cfg := &config.Config{
		Env:            "test",
		DatabaseType:   "sqlite",
		DatabaseURL:    "test.db",
		JWTSecret:      "test-secret",
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		AllowedOrigins: []string{"http://localhost:3000"},
		JWTType:        "HS256",
	}

	return cfg
}

// initTestSetup initializes the test setup
func initTestSetup(t *testing.T, cfg *config.Config) *testSetup {
	// Initialize logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

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
		Log: &logger,
	})
	require.NoError(t, err)

	// Create dependencies struct
	gqlDeps := &graphql.Dependencies{
		Log:                   &logger,
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
		AuthenticatorProvider: authProvider,
		EmailProvider:         emailProvider,
		EventsProvider:        eventsProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		SMSProvider:           smsProvider,
		StorageProvider:       storageProvider,
		TokenProvider:         tokenProvider,
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
