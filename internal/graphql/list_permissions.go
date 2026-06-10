package graphql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// ListPermissions enumerates the fully-qualified object ids of object_type the
// subject holds relation on — "which documents can I view?". Ideal for
// filtering list pages down to what the subject may access.
//
// SUBJECT TRUST GATE: same rules as CheckPermissions (token subject by
// default; explicit `user` for super-admins or self). The result set is
// capped: listing is an expensive enumeration surface.
// Permission: authorized user.
func (g *graphqlProvider) ListPermissions(ctx context.Context, params *model.ListPermissionsInput) (*model.ListPermissionsResponse, error) {
	log := g.Log.With().Str("func", "ListPermissions").Logger()
	if g.AuthzEngine == nil {
		return nil, errFgaNotEnabled
	}
	if params == nil || strings.TrimSpace(params.Relation) == "" || strings.TrimSpace(params.ObjectType) == "" {
		return nil, fmt.Errorf("relation and object_type are required")
	}
	subject, err := g.resolveFgaSubject(ctx, refs.StringValue(params.User))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve subject")
		return nil, err
	}
	start := time.Now()
	objects, err := g.AuthzEngine.ListObjects(ctx, subject, params.Relation, params.ObjectType)
	metrics.ObserveFgaCheckDuration(metrics.FgaOpListPermissions, time.Since(start).Seconds())
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpListPermissions, metrics.FgaResultError)
		log.Debug().Err(err).Msg("ListPermissions failed; denying")
		return nil, fmt.Errorf("authorization list failed")
	}
	metrics.RecordFgaOperation(metrics.FgaOpListPermissions, metrics.FgaResultSuccess)
	// Cap the result set; ListObjects is an expensive enumeration surface.
	if len(objects) > maxFgaListResults {
		objects = objects[:maxFgaListResults]
	}
	return &model.ListPermissionsResponse{Objects: objects}, nil
}
