package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AuditLogs is the method to list audit logs.
// Permission: authorizer:admin
func (g *graphqlProvider) AuditLogs(ctx context.Context, params *model.ListAuditLogRequest) (*model.AuditLogs, error) {
	log := g.Log.With().Str("func", "AuditLogs").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	var pagination *model.Pagination
	filter := make(map[string]interface{})

	if params != nil {
		pagination = utils.GetPagination(&model.PaginatedRequest{
			Pagination: params.Pagination,
		})
		if refs.StringValue(params.Action) != "" {
			filter["action"] = refs.StringValue(params.Action)
		}
		if refs.StringValue(params.ActorID) != "" {
			filter["actor_id"] = refs.StringValue(params.ActorID)
		}
		if refs.StringValue(params.ResourceType) != "" {
			filter["resource_type"] = refs.StringValue(params.ResourceType)
		}
		if refs.StringValue(params.ResourceID) != "" {
			filter["resource_id"] = refs.StringValue(params.ResourceID)
		}
		if params.FromTimestamp != nil {
			filter["from_timestamp"] = *params.FromTimestamp
		}
		if params.ToTimestamp != nil {
			filter["to_timestamp"] = *params.ToTimestamp
		}
	} else {
		pagination = utils.GetPagination(nil)
	}

	auditLogs, paginationRes, err := g.StorageProvider.ListAuditLogs(ctx, pagination, filter)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListAuditLogs")
		return nil, err
	}
	resItems := make([]*model.AuditLog, len(auditLogs))
	for i, auditLog := range auditLogs {
		resItems[i] = auditLog.AsAPIAuditLog()
	}
	return &model.AuditLogs{
		Pagination: paginationRes,
		AuditLogs:  resItems,
	}, nil
}
