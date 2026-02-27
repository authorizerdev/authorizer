package graph

import (
	"github.com/authorizerdev/authorizer/internal/graphql"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	GraphQLProvider graphql.Provider
}
