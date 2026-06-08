package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// maxFgaListResults caps the number of objects returned by fga_list_objects and
// the page size of admin tuple reads. ListObjects is an expensive enumeration
// surface, so the result set is bounded.
const maxFgaListResults = 1000

// maxFgaBatchChecks caps the number of pairs accepted in a single batch check.
const maxFgaBatchChecks = 100

// principalForRequest resolves the authenticated caller and returns the pinned
// OpenFGA subject ("user:<sub>"). The principal is ALWAYS derived from the auth
// token / session — never from client input — so a caller can only ask about
// their own access.
func (g *graphqlProvider) principalForRequest(ctx context.Context) (string, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return "", err
	}
	tokenData, err := g.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenData.UserID) == "" {
		return "", fmt.Errorf("unauthorized")
	}
	return "user:" + tokenData.UserID, nil
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
	// PRINCIPAL PINNING — derive subject from the authenticated caller only.
	principal, err := g.principalForRequest(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve principal")
		return nil, fmt.Errorf("unauthorized")
	}
	ctxTuples, err := toContextualTuples(params.ContextualTuples)
	if err != nil {
		return nil, err
	}
	allowed, err := g.AuthzEngine.Check(ctx, principal, params.Relation, params.Object, ctxTuples...)
	if err != nil {
		// Fail closed: treat engine error as deny.
		log.Debug().Err(err).Msg("Check failed; denying")
		return nil, fmt.Errorf("authorization check failed")
	}
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
	// PRINCIPAL PINNING — derive subject from the authenticated caller only.
	principal, err := g.principalForRequest(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve principal")
		return nil, fmt.Errorf("unauthorized")
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
			User:             principal,
			Relation:         c.Relation,
			Object:           c.Object,
			ContextualTuples: ctxTuples,
		})
	}
	results, err := g.AuthzEngine.BatchCheck(ctx, requests)
	if err != nil {
		// Fail closed for the whole batch.
		log.Debug().Err(err).Msg("BatchCheck failed; denying")
		return nil, fmt.Errorf("authorization check failed")
	}
	out := &model.FgaBatchCheckResponse{Results: make([]*model.FgaCheckResponse, 0, len(results))}
	for _, r := range results {
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
	// PRINCIPAL PINNING — derive subject from the authenticated caller only.
	principal, err := g.principalForRequest(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve principal")
		return nil, fmt.Errorf("unauthorized")
	}
	objects, err := g.AuthzEngine.ListObjects(ctx, principal, params.Relation, params.ObjectType)
	if err != nil {
		log.Debug().Err(err).Msg("ListObjects failed; denying")
		return nil, fmt.Errorf("authorization list failed")
	}
	// Cap the result set; ListObjects is an expensive enumeration surface.
	if len(objects) > maxFgaListResults {
		objects = objects[:maxFgaListResults]
	}
	return &model.FgaListObjectsResponse{Objects: objects}, nil
}
