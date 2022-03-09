package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/sessionstore"
)

// RevokeResolver resolver to revoke refresh token
func RevokeResolver(ctx context.Context, params model.OAuthRevokeInput) (*model.Response, error) {
	sessionstore.RemoveState(params.RefreshToken)
	return &model.Response{
		Message: "Token revoked",
	}, nil
}
