package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// User delegates to the transport-agnostic service layer. Resolver is a thin
// transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) User(ctx context.Context, params *model.GetUserRequest) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().User(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// EnrolledMFAMethods backs the lazily-resolved User.enrolled_mfa_methods field.
// It is only invoked when a query selects that field, so it never runs on the
// login/signup/profile paths that don't ask for it.
//
// No auth guard here: the field takes no argument and only ever resolves the
// already-authorized parent User (self via profile/session, or an admin via
// _users/_user, which are guarded at the parent query). There is no
// user-supplied ID and thus no account-enumeration vector.
//
// Permissions: inherited from the parent User query.
func (g *graphqlProvider) EnrolledMFAMethods(ctx context.Context, user *model.User) ([]string, error) {
	if user == nil {
		return []string{}, nil
	}
	return g.ServiceProvider.EnrolledMFAMethods(ctx, user.ID)
}
