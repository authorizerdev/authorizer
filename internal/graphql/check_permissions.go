package graphql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// CheckPermissions evaluates one or more permission checks ("does the subject
// have <relation> on <object>?") in a single call and returns one result per
// check, in order. A single check is simply a list of one.
//
// SUBJECT TRUST GATE: the subject defaults to the authenticated caller's token
// subject; an explicit `user` is honored only for super-admins or when it
// equals the caller's own subject (see resolveFgaSubject). Fail-closed: any
// engine error denies.
// Permission: authorized user.
func (g *graphqlProvider) CheckPermissions(ctx context.Context, params *model.CheckPermissionsInput) (*model.CheckPermissionsResponse, error) {
	log := g.Log.With().Str("func", "CheckPermissions").Logger()
	if g.AuthzEngine == nil {
		return nil, errFgaNotEnabled
	}
	if params == nil || len(params.Checks) == 0 {
		return nil, fmt.Errorf("at least one check is required")
	}
	if len(params.Checks) > maxPermissionChecks {
		return nil, fmt.Errorf("too many checks: max %d per request", maxPermissionChecks)
	}
	subject, err := g.resolveFgaSubject(ctx, refs.StringValue(params.User))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve subject")
		return nil, err
	}
	requests := make([]engine.CheckRequest, 0, len(params.Checks))
	for _, c := range params.Checks {
		if c == nil || strings.TrimSpace(c.Relation) == "" || strings.TrimSpace(c.Object) == "" {
			return nil, fmt.Errorf("each check requires relation and object")
		}
		ctxTuples, err := toContextualTuples(c.ContextualTuples)
		if err != nil {
			return nil, err
		}
		requests = append(requests, engine.CheckRequest{
			User:             subject,
			Relation:         c.Relation,
			Object:           c.Object,
			ContextualTuples: ctxTuples,
		})
	}
	start := time.Now()
	results, err := g.AuthzEngine.BatchCheck(ctx, requests)
	metrics.ObserveFgaCheckDuration(metrics.FgaOpCheckPermissions, time.Since(start).Seconds())
	if err != nil {
		// Fail closed for the whole call.
		metrics.RecordFgaCheck(metrics.FgaOpCheckPermissions, metrics.FgaResultError)
		log.Debug().Err(err).Msg("CheckPermissions failed; denying")
		return nil, fmt.Errorf("authorization check failed")
	}
	out := &model.CheckPermissionsResponse{Results: make([]*model.PermissionCheckResult, 0, len(results))}
	for i, r := range results {
		// Record each decision so adoption/denial rates reflect every pair.
		metrics.RecordFgaCheckResult(metrics.FgaOpCheckPermissions, r.Allowed)
		out.Results = append(out.Results, &model.PermissionCheckResult{
			Relation: params.Checks[i].Relation,
			Object:   params.Checks[i].Object,
			Allowed:  r.Allowed,
		})
	}
	return out, nil
}
