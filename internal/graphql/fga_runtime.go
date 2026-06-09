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
	"github.com/authorizerdev/authorizer/internal/utils"
)

// maxFgaListResults caps the number of objects returned by fga_list_objects and
// the page size of admin tuple reads. ListObjects is an expensive enumeration
// surface, so the result set is bounded.
const maxFgaListResults = 1000

// maxFgaBatchChecks caps the number of pairs accepted in a single batch check.
const maxFgaBatchChecks = 100

// resolveFgaSubject is the single, centralized trust gate for the FGA decision
// ops (fga_check, fga_batch_check, fga_list_objects). It decides which OpenFGA
// subject ("type:id") a decision is evaluated for, given the optional
// client-supplied explicitUser override.
//
// Rules (fail-closed):
//   - The trust level is ALWAYS derived from the auth token/session/admin cookie
//     — never from client input.
//   - super-admin caller: an explicitly supplied explicitUser is honored
//     (validated to look like "type:id"); if absent it defaults to the
//     super-admin's own "user:<sub>" when they also carry a user token, else it
//     is required (a bare admin cookie has no subject of its own).
//   - non-super-admin caller (ordinary end-user token/cookie):
//   - explicitUser empty  → pin to the caller's own "user:<sub>".
//   - explicitUser set    → REJECT. We do not silently honor it (would let an
//     end user enumerate another subject's access — IDOR / info-disclosure)
//     and we do not silently ignore it.
//
// TODO(phase-2 M2M): machine-to-machine / client-credentials callers should also
// be allowed to pass an explicit user once that caller type exists; extend the
// trust check here (the rule must stay centralized in this one helper).
func (g *graphqlProvider) resolveFgaSubject(ctx context.Context, explicitUser string) (string, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return "", err
	}
	explicitUser = strings.TrimSpace(explicitUser)

	// Determine the trust level first. Super-admin authenticates via the admin
	// cookie/secret and need not carry a user token of its own, so this check
	// must not depend on resolving a user subject.
	isSuperAdmin := g.TokenProvider.IsSuperAdmin(gc)

	if explicitUser != "" {
		// A subject override was requested — only super-admin callers may target
		// another subject. Everyone else is rejected (no silent self-fallback, no
		// silent ignore).
		if !isSuperAdmin {
			return "", fmt.Errorf("not authorized to query authorization for another subject")
		}
		if err := validateFgaSubject(explicitUser); err != nil {
			return "", err
		}
		return explicitUser, nil
	}

	// No override: pin to the caller's own subject (unchanged self-pinning
	// behavior). This requires a resolvable user subject from the token/session.
	tokenData, err := g.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil || strings.TrimSpace(tokenData.UserID) == "" {
		return "", fmt.Errorf("unauthorized")
	}
	return "user:" + tokenData.UserID, nil
}

// validateFgaSubject ensures a super-admin-supplied subject override is in
// OpenFGA "type:id" form (both halves non-empty). It rejects usersets
// ("type:id#relation") and malformed values.
func validateFgaSubject(user string) error {
	objType, objID, found := strings.Cut(user, ":")
	if !found || strings.TrimSpace(objType) == "" || strings.TrimSpace(objID) == "" {
		return fmt.Errorf("user must be in type:id form, got %q", user)
	}
	if strings.Contains(objID, "#") {
		return fmt.Errorf("user must be a concrete subject in type:id form, not a userset, got %q", user)
	}
	return nil
}

// toContextualTuples converts client-supplied contextual tuples. These are
// request-scoped only (never persisted) and are safe to accept from the client.
func toContextualTuples(in []*model.FgaTupleInput) ([]engine.ContextualTuple, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]engine.ContextualTuple, 0, len(in))
	for _, t := range in {
		if t == nil || strings.TrimSpace(t.User) == "" || strings.TrimSpace(t.Relation) == "" || strings.TrimSpace(t.Object) == "" {
			return nil, fmt.Errorf("each contextual tuple requires user, relation and object")
		}
		out = append(out, engine.ContextualTuple{User: t.User, Relation: t.Relation, Object: t.Object})
	}
	return out, nil
}

// FgaCheck answers "is the authenticated caller related to object via relation?".
// PRINCIPAL PINNING: the subject is the caller's token sub ("user:<sub>"), never
// client input. Fail-closed: any engine error denies.
// Permission: authorized user.
func (g *graphqlProvider) FgaCheck(ctx context.Context, params *model.FgaCheckInput) (*model.FgaCheckResponse, error) {
	log := g.Log.With().Str("func", "FgaCheck").Logger()
	if g.AuthzEngine == nil {
		return nil, errFgaNotEnabled
	}
	if params == nil || strings.TrimSpace(params.Relation) == "" || strings.TrimSpace(params.Object) == "" {
		return nil, fmt.Errorf("relation and object are required")
	}
	// TRUST GATE — derive subject from the authenticated caller; an explicit
	// `user` override is honored only for super-admins (see resolveFgaSubject).
	principal, err := g.resolveFgaSubject(ctx, refs.StringValue(params.User))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve subject")
		return nil, err
	}
	ctxTuples, err := toContextualTuples(params.ContextualTuples)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	allowed, err := g.AuthzEngine.Check(ctx, principal, params.Relation, params.Object, ctxTuples...)
	metrics.ObserveFgaCheckDuration(metrics.FgaOpCheck, time.Since(start).Seconds())
	if err != nil {
		// Fail closed: treat engine error as deny.
		metrics.RecordFgaCheck(metrics.FgaOpCheck, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Check failed; denying")
		return nil, fmt.Errorf("authorization check failed")
	}
	metrics.RecordFgaCheckResult(metrics.FgaOpCheck, allowed)
	return &model.FgaCheckResponse{Allowed: allowed}, nil
}

// FgaBatchCheck evaluates multiple relation/object pairs for the authenticated
// caller. Principal pinned; fail-closed per the engine contract.
// Permission: authorized user.
func (g *graphqlProvider) FgaBatchCheck(ctx context.Context, params *model.FgaBatchCheckInput) (*model.FgaBatchCheckResponse, error) {
	log := g.Log.With().Str("func", "FgaBatchCheck").Logger()
	if g.AuthzEngine == nil {
		return nil, errFgaNotEnabled
	}
	if params == nil || len(params.Checks) == 0 {
		return nil, fmt.Errorf("at least one check is required")
	}
	if len(params.Checks) > maxFgaBatchChecks {
		return nil, fmt.Errorf("too many checks: max %d per request", maxFgaBatchChecks)
	}
	requests := make([]engine.CheckRequest, 0, len(params.Checks))
	for _, c := range params.Checks {
		if c == nil || strings.TrimSpace(c.Relation) == "" || strings.TrimSpace(c.Object) == "" {
			return nil, fmt.Errorf("each check requires relation and object")
		}
		// TRUST GATE — derive subject per item from the authenticated caller; an
		// explicit `user` override is honored only for super-admins.
		principal, err := g.resolveFgaSubject(ctx, refs.StringValue(c.User))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to resolve subject")
			return nil, err
		}
		ctxTuples, err := toContextualTuples(c.ContextualTuples)
		if err != nil {
			return nil, err
		}
		requests = append(requests, engine.CheckRequest{
			User:             principal,
			Relation:         c.Relation,
			Object:           c.Object,
			ContextualTuples: ctxTuples,
		})
	}
	start := time.Now()
	results, err := g.AuthzEngine.BatchCheck(ctx, requests)
	metrics.ObserveFgaCheckDuration(metrics.FgaOpBatchCheck, time.Since(start).Seconds())
	if err != nil {
		// Fail closed for the whole batch.
		metrics.RecordFgaCheck(metrics.FgaOpBatchCheck, metrics.FgaResultError)
		log.Debug().Err(err).Msg("BatchCheck failed; denying")
		return nil, fmt.Errorf("authorization check failed")
	}
	out := &model.FgaBatchCheckResponse{Results: make([]*model.FgaCheckResponse, 0, len(results))}
	for _, r := range results {
		// Record each sub-decision so adoption/denial rates reflect every pair.
		metrics.RecordFgaCheckResult(metrics.FgaOpBatchCheck, r.Allowed)
		out.Results = append(out.Results, &model.FgaCheckResponse{Allowed: r.Allowed})
	}
	return out, nil
}

// FgaListObjects enumerates objects of object_type the authenticated caller
// relates to via relation. Principal pinned; result set capped (enumeration
// surface). Fail-closed.
// Permission: authorized user.
func (g *graphqlProvider) FgaListObjects(ctx context.Context, params *model.FgaListObjectsInput) (*model.FgaListObjectsResponse, error) {
	log := g.Log.With().Str("func", "FgaListObjects").Logger()
	if g.AuthzEngine == nil {
		return nil, errFgaNotEnabled
	}
	if params == nil || strings.TrimSpace(params.Relation) == "" || strings.TrimSpace(params.ObjectType) == "" {
		return nil, fmt.Errorf("relation and object_type are required")
	}
	// TRUST GATE — derive subject from the authenticated caller; an explicit
	// `user` override is honored only for super-admins (see resolveFgaSubject).
	principal, err := g.resolveFgaSubject(ctx, refs.StringValue(params.User))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve subject")
		return nil, err
	}
	start := time.Now()
	objects, err := g.AuthzEngine.ListObjects(ctx, principal, params.Relation, params.ObjectType)
	metrics.ObserveFgaCheckDuration(metrics.FgaOpListObjects, time.Since(start).Seconds())
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpListObjects, metrics.FgaResultError)
		log.Debug().Err(err).Msg("ListObjects failed; denying")
		return nil, fmt.Errorf("authorization list failed")
	}
	metrics.RecordFgaOperation(metrics.FgaOpListObjects, metrics.FgaResultSuccess)
	// Cap the result set; ListObjects is an expensive enumeration surface.
	if len(objects) > maxFgaListResults {
		objects = objects[:maxFgaListResults]
	}
	return &model.FgaListObjectsResponse{Objects: objects}, nil
}
