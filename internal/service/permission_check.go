package service

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authorization"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// enforceRequiredPermissions evaluates each required permission against the
// authorization provider with AND semantics — every entry must be allowed,
// otherwise the caller is treated as unauthorized. Direct port of the
// graphqlProvider helper of the same name; the metrics + invariants
// (one terminal return per call, exactly one metric emission) are preserved.
//
// endpoint identifies the operation that called this helper and becomes
// the `endpoint` label on authorizer_required_permissions_checks_total;
// it must be one of metrics.RequiredPermissionsEndpoint* to avoid
// cardinality explosion.
func (p *provider) enforceRequiredPermissions(
	ctx context.Context,
	log zerolog.Logger,
	endpoint string,
	userID string,
	roles []string,
	required []*model.PermissionInput,
) error {
	if len(required) == 0 {
		metrics.RecordRequiredPermissionsCheck(endpoint, metrics.RequiredPermissionsOutcomeNotRequested)
		return nil
	}
	principal := &authorization.Principal{
		ID:    userID,
		Type:  constants.PrincipalTypeUser,
		Roles: roles,
	}
	for _, pi := range required {
		if pi == nil {
			continue
		}
		res, err := p.AuthorizationProvider.CheckPermission(ctx, principal, pi.Resource, pi.Scope)
		if err != nil {
			log.Debug().Err(err).Str("resource", pi.Resource).Str("scope", pi.Scope).Msg("required permission check errored")
			metrics.RecordRequiredPermissionsCheck(endpoint, metrics.RequiredPermissionsOutcomeError)
			return PermissionDenied("unauthorized")
		}
		if res == nil || !res.Allowed {
			log.Debug().Str("resource", pi.Resource).Str("scope", pi.Scope).Msg("required permission denied")
			metrics.RecordRequiredPermissionsCheck(endpoint, metrics.RequiredPermissionsOutcomeDenied)
			return PermissionDenied("unauthorized")
		}
	}
	metrics.RecordRequiredPermissionsCheck(endpoint, metrics.RequiredPermissionsOutcomeGranted)
	return nil
}
