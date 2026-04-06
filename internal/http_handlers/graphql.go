package http_handlers

import (
	"context"
	"net/http"
	"sort"
	"sync"
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
	"github.com/authorizerdev/authorizer/internal/utils"
)

type gqlResolvedFieldsCtxKey struct{}

// resolvedFieldsCollector gathers unique GraphQL field names for one operation.
type resolvedFieldsCollector struct {
	mu     sync.Mutex
	fields map[string]struct{}
}

func (c *resolvedFieldsCollector) add(name string) {
	if name == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fields == nil {
		c.fields = make(map[string]struct{})
	}
	c.fields[name] = struct{}{}
}

func (c *resolvedFieldsCollector) sortedUnique() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.fields))
	for f := range c.fields {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// gqlCollectResolvedFieldsMiddleware records each resolved field name into the per-operation collector.
func (*httpProvider) gqlCollectResolvedFieldsMiddleware() gql.FieldMiddleware {
	return func(ctx context.Context, next gql.Resolver) (interface{}, error) {
		if col, ok := ctx.Value(gqlResolvedFieldsCtxKey{}).(*resolvedFieldsCollector); ok && col != nil {
			if fc := gql.GetFieldContext(ctx); fc != nil && fc.Field.Field != nil {
				col.add(fc.Field.Name)
			}
		}
		return next(ctx)
	}
}

// gqlMetricsMiddleware records GraphQL operation duration and errors.
// It captures errors returned in HTTP 200 responses (GraphQL convention).
func (h *httpProvider) gqlMetricsMiddleware() gql.OperationMiddleware {
	return func(ctx context.Context, next gql.OperationHandler) gql.ResponseHandler {
		operationName := ""
		if oc := gql.GetOperationContext(ctx); oc != nil {
			operationName = oc.OperationName
		}
		opMetricLabel := metrics.GraphQLOperationPrometheusLabel(operationName)
		start := time.Now()

		collector := &resolvedFieldsCollector{}
		ctx = context.WithValue(ctx, gqlResolvedFieldsCtxKey{}, collector)

		responseHandler := next(ctx)

		return func(ctx context.Context) *gql.Response {
			resp := responseHandler(ctx)
			fields := collector.sortedUnique()
			if resp == nil {
				h.Dependencies.Log.Warn().
					Str("operation", operationName).
					Str("operation_metric_label", opMetricLabel).
					Strs("resolved_fields", fields).
					Msg("GraphQL operation returned no response")
				return resp
			}
			duration := time.Since(start).Seconds()
			metrics.GraphQLRequestDuration.WithLabelValues(opMetricLabel).Observe(duration)

			if len(resp.Errors) > 0 {
				metrics.RecordGraphQLError(operationName)
			}
			logEvt := h.Dependencies.Log.Info().
				Str("operation", operationName).
				Str("operation_metric_label", opMetricLabel).
				Int("resolved_field_count", len(fields))
			logEvt.Msg("GraphQL operation completed")
			h.Dependencies.Log.Debug().
				Str("operation", operationName).
				Strs("resolved_fields", fields).
				Msg("GraphQL resolved fields")
			return resp
		}
	}
}

// GraphqlHandler is the main handler that handles all GraphQL requests.
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
		return func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":             "graphql_unavailable",
				"error_description": "GraphQL service failed to initialize.",
			})
		}
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
	srv.AroundFields(h.gqlCollectResolvedFieldsMiddleware())
	srv.AroundOperations(h.gqlMetricsMiddleware())
	if h.Config.EnableGraphQLIntrospection {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})
	// Limit query complexity to prevent resource exhaustion
	srv.Use(extension.FixedComplexityLimit(300))

	return func(c *gin.Context) {
		// Create a custom handler that ensures gin context is available
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ensure the gin context is available in the request context
			ctx := utils.ContextWithGin(r.Context(), c)
			r = r.WithContext(ctx)
			srv.ServeHTTP(w, r)
		})
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
