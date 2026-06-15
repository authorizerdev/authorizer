package service

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Audit log filter keys passed to StorageProvider.ListAuditLogs. Centralized as
// constants so the service layer and storage providers agree on the contract.
const (
	auditFilterAction        = "action"
	auditFilterActorID       = "actor_id"
	auditFilterResourceType  = "resource_type"
	auditFilterResourceID    = "resource_id"
	auditFilterFromTimestamp = "from_timestamp"
	auditFilterToTimestamp   = "to_timestamp"
)

// AuditLogs returns a paginated, optionally-filtered list of audit log entries.
// Requires super-admin auth. Logic migrated from internal/graphql/audit_logs.go.
func (p *provider) AuditLogs(ctx context.Context, meta RequestMetadata, params *model.ListAuditLogRequest) (*model.AuditLogs, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AuditLogs").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	var pagination *model.Pagination
	filter := make(map[string]interface{})

	if params != nil {
		pagination = utils.GetPagination(&model.PaginatedRequest{
			Pagination: params.Pagination,
		})
		if refs.StringValue(params.Action) != "" {
			filter[auditFilterAction] = refs.StringValue(params.Action)
		}
		if refs.StringValue(params.ActorID) != "" {
			filter[auditFilterActorID] = refs.StringValue(params.ActorID)
		}
		if refs.StringValue(params.ResourceType) != "" {
			filter[auditFilterResourceType] = refs.StringValue(params.ResourceType)
		}
		if refs.StringValue(params.ResourceID) != "" {
			filter[auditFilterResourceID] = refs.StringValue(params.ResourceID)
		}
		if params.FromTimestamp != nil {
			filter[auditFilterFromTimestamp] = *params.FromTimestamp
		}
		if params.ToTimestamp != nil {
			filter[auditFilterToTimestamp] = *params.ToTimestamp
		}
	} else {
		pagination = utils.GetPagination(nil)
	}

	auditLogs, paginationRes, err := p.StorageProvider.ListAuditLogs(ctx, pagination, filter)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListAuditLogs")
		return nil, nil, err
	}
	resItems := make([]*model.AuditLog, len(auditLogs))
	for i, auditLog := range auditLogs {
		resItems[i] = auditLog.AsAPIAuditLog()
	}
	return &model.AuditLogs{
		Pagination: paginationRes,
		AuditLogs:  resItems,
	}, nil, nil
}
