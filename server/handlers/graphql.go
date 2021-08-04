package handlers

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/authorizerdev/authorizer/server/graph"
	"github.com/authorizerdev/authorizer/server/graph/generated"
	"github.com/gin-gonic/gin"
)

// Defining the Graphql handler
func GraphqlHandler() gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
