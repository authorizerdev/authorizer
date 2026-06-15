package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AuditLogs delegates to the transport-agnostic service layer. Resolver is a
// thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) AuditLogs(ctx context.Context, params *model.ListAuditLogRequest) (*model.AuditLogs, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().AuditLogs(ctx, service.MetaFromGin(gc), params)
	return res, err
}
