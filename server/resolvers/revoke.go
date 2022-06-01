package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

// RevokeResolver resolver to revoke refresh token
func RevokeResolver(ctx context.Context, params model.OAuthRevokeInput) (*model.Response, error) {
	memorystore.Provider.RemoveState(params.RefreshToken)
	return &model.Response{
		Message: "Token revoked",
	}, nil
}
