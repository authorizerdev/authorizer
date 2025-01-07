package http_handlers

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/authorizerdev/authorizer/internal/graph"
	"github.com/authorizerdev/authorizer/internal/graph/generated"
	"github.com/authorizerdev/authorizer/internal/graphql"
	"github.com/gin-gonic/gin"
)

// GraphqlHandler is the main handler that handels all the graphql requests
func (h *httpProvider) GraphqlHandler() gin.HandlerFunc {
	gqlProvider, err := graphql.New(h.Config, &graphql.Dependencies{
		Log:                   h.Log,
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
	gqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		GraphQLProvider: gqlProvider,
	}}))

	return func(c *gin.Context) {
		gqlHandler.ServeHTTP(c.Writer, c.Request)
	}
}
