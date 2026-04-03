package http_handlers

import (
	"context"
	"net/http"
	"time"

	gql "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gin-gonic/gin"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/authorizerdev/authorizer/internal/graph"
	"github.com/authorizerdev/authorizer/internal/graph/generated"
	"github.com/authorizerdev/authorizer/internal/graphql"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

func (h *httpProvider) gqlLoggingMiddleware() gql.FieldMiddleware {
	return func(ctx context.Context, next gql.Resolver) (res interface{}, err error) {
		// Get details of the current operation
		oc := gql.GetOperationContext(ctx)
		field := gql.GetFieldContext(ctx)

		// Log operation details
		h.Dependencies.Log.Info().
			Str("operation", oc.OperationName).
			Str("query", field.Field.Name).
			// Interface("arguments", field.Args). // Enable only for debugging purpose else sensitive data will be logged
			Msg("GraphQL field resolved")

		// Call the next resolver
		return next(ctx)
	}
}

// gqlMetricsMiddleware records GraphQL operation duration and errors.
// It captures errors returned in HTTP 200 responses (GraphQL convention).
func (h *httpProvider) gqlMetricsMiddleware() gql.OperationMiddleware {
	return func(ctx context.Context, next gql.OperationHandler) gql.ResponseHandler {
		oc := gql.GetOperationContext(ctx)
		operationName := oc.OperationName
		if operationName == "" {
			operationName = "anonymous"
		}
		start := time.Now()

		responseHandler := next(ctx)

		return func(ctx context.Context) *gql.Response {
			resp := responseHandler(ctx)
			if resp != nil {
				duration := time.Since(start).Seconds()
				metrics.GraphQLRequestDuration.WithLabelValues(operationName).Observe(duration)

				if len(resp.Errors) > 0 {
					metrics.RecordGraphQLError(operationName)
				}
			}
			return resp
		}
	}
}

// GraphqlHandler is the main handler that handels all the graphql requests
func (h *httpProvider) GraphqlHandler() gin.HandlerFunc {
	gqlProvider, err := graphql.New(h.Config, &graphql.Dependencies{
		Log:                   h.Log,
		AuditProvider:         h.AuditProvider,
		AuthenticatorProvider: h.AuthenticatorProvider,
		EmailProvider:         h.EmailProvider,
		EventsProvider:        h.EventsProvider,
		MemoryStoreProvider:   h.MemoryStoreProvider,
		SMSProvider:           h.SMSProvider,
		StorageProvider:       h.StorageProvider,
		TokenProvider:         h.TokenProvider,
	})
	if err != nil {
		h.Log.Error().Err(err).Msg("Failed to create graphql provider")
		return nil
	}

	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		GraphQLProvider: gqlProvider,
	}}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.AroundFields(h.gqlLoggingMiddleware())
	srv.AroundOperations(h.gqlMetricsMiddleware())
	if h.Config.EnableGraphQLIntrospection {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	return func(c *gin.Context) {
		// Create a custom handler that ensures gin context is available
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ensure the gin context is available in the request context
			ctx := context.WithValue(r.Context(), "GinContextKey", c)
			r = r.WithContext(ctx)
			srv.ServeHTTP(w, r)
		})
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
