package graphql

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authorization"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// enforceRequiredPermissions evaluates each required permission against the
// authorization provider with AND semantics — every entry must be allowed,
// otherwise the caller is treated as unauthorized.
//
// endpoint identifies the GraphQL operation that called this helper and
// becomes the `endpoint` label on authorizer_required_permissions_checks_total.
// It must be one of metrics.RequiredPermissionsEndpoint* — passing an
// unbounded string risks Prometheus cardinality explosion.
//
// When required is empty (the common case) the metric is incremented with
// outcome=not_requested and the helper returns nil so existing callers see no
// behavior change.
func (g *graphqlProvider) enforceRequiredPermissions(
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
	for _, p := range required {
		if p == nil {
			continue
		}
		res, err := g.AuthorizationProvider.CheckPermission(ctx, principal, p.Resource, p.Scope)
		if err != nil {
			log.Debug().Err(err).Str("resource", p.Resource).Str("scope", p.Scope).Msg("required permission check errored")
			metrics.RecordRequiredPermissionsCheck(endpoint, metrics.RequiredPermissionsOutcomeError)
			return errors.New("unauthorized")
		}
		if res == nil || !res.Allowed {
			log.Debug().Str("resource", p.Resource).Str("scope", p.Scope).Msg("required permission denied")
			metrics.RecordRequiredPermissionsCheck(endpoint, metrics.RequiredPermissionsOutcomeDenied)
			return errors.New("unauthorized")
		}
	}
	metrics.RecordRequiredPermissionsCheck(endpoint, metrics.RequiredPermissionsOutcomeGranted)
	return nil
}
