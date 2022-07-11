package resolvers

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

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
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)

	res, err := db.Provider.ListVerificationRequests(ctx, pagination)
	if err != nil {
		log.Debug("Failed to get verification requests: ", err)
		return nil, err
	}

	return res, nil
}
