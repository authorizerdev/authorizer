package handlers

import (
	"github.com/99designs/gqlgen/graphql/handler"
	graph "github.com/authorizerdev/authorizer/server/graph"
	"github.com/authorizerdev/authorizer/server/graph/generated"
	"github.com/gin-gonic/gin"
)

// GraphqlHandler is the main handler that handels all the graphql requests
func GraphqlHandler() gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
