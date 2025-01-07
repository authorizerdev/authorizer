package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Env is a resolver for config query
// Permissions: authorizer:admin
// Deprecated: this is deprecated
func (g *graphqlProvider) Env(ctx context.Context) (*model.Env, error) {
	return nil, fmt.Errorf("deprecated")
}
