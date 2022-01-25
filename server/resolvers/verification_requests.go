package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerificationRequestsResolver is a resolver for verification requests query
// This is admin only query
func VerificationRequestsResolver(ctx context.Context, params *model.PaginatedInput) (*model.VerificationRequests, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)

	res, err := db.Provider.ListVerificationRequests(pagination)
	if err != nil {
		return nil, err
	}

	return res, nil
}
