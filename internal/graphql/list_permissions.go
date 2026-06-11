package graphql

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// maxConcurrentFgaListCalls bounds the parallel ListObjects expansions issued
// by an unfiltered list_permissions call so one request cannot saturate the
// embedded engine.
const maxConcurrentFgaListCalls = 5

// typeRelation is one (object type, relation) pair to enumerate.
type typeRelation struct {
	objType  string
	relation string
}

// ListPermissions enumerates what the subject can access. With both filters
// set it answers "which <object_type>s can I <relation>?" via a single
// ListObjects call. When either filter is omitted, every matching (type,
// relation) pair of the active model is enumerated with bounded concurrency —
// an empty input returns ALL permissions the subject holds.
//
// SUBJECT TRUST GATE: same rules as CheckPermissions (token subject by
// default; explicit `user` for super-admins or self). The result set is
// capped at maxFgaListResults and `truncated` reports when the cap was hit:
// listing is an expensive enumeration surface.
// Permission: authorized user.
func (g *graphqlProvider) ListPermissions(ctx context.Context, params *model.ListPermissionsInput) (*model.ListPermissionsResponse, error) {
	log := g.Log.With().Str("func", "ListPermissions").Logger()
	if g.AuthzEngine == nil {
		return nil, errFgaNotEnabled
	}
	if params == nil {
		params = &model.ListPermissionsInput{}
	}
	relationFilter := strings.TrimSpace(refs.StringValue(params.Relation))
	typeFilter := strings.TrimSpace(refs.StringValue(params.ObjectType))
	subject, err := g.resolveFgaSubject(ctx, refs.StringValue(params.User))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve subject")
		return nil, err
	}

	start := time.Now()
	pairs, err := g.listPermissionPairs(ctx, relationFilter, typeFilter)
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpListPermissions, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to resolve model type relations; denying")
		return nil, fmt.Errorf("authorization list failed")
	}

	// Enumerate each pair with bounded concurrency; results stay positionally
	// aligned with pairs so aggregation order is deterministic.
	results := make([][]string, len(pairs))
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(maxConcurrentFgaListCalls)
	for i, p := range pairs {
		eg.Go(func() error {
			objects, lerr := g.AuthzEngine.ListObjects(egCtx, subject, p.relation, p.objType)
			if lerr != nil {
				return lerr
			}
			results[i] = objects
			return nil
		})
	}
	egErr := eg.Wait()
	metrics.ObserveFgaCheckDuration(metrics.FgaOpListPermissions, time.Since(start).Seconds())
	if egErr != nil {
		metrics.RecordFgaOperation(metrics.FgaOpListPermissions, metrics.FgaResultError)
		log.Debug().Err(egErr).Msg("ListPermissions failed; denying")
		return nil, fmt.Errorf("authorization list failed")
	}
	metrics.RecordFgaOperation(metrics.FgaOpListPermissions, metrics.FgaResultSuccess)

	// Aggregate under the global cap; `truncated` tells callers more exist.
	permissions := make([]*model.Permission, 0)
	objects := make([]string, 0)
	seen := make(map[string]struct{})
	truncated := false
	for i, objs := range results {
		for _, obj := range objs {
			if len(permissions) >= maxFgaListResults {
				truncated = true
				break
			}
			permissions = append(permissions, &model.Permission{Object: obj, Relation: pairs[i].relation})
			if _, ok := seen[obj]; !ok {
				seen[obj] = struct{}{}
				objects = append(objects, obj)
			}
		}
		if truncated {
			break
		}
	}
	return &model.ListPermissionsResponse{
		Objects:     objects,
		Permissions: permissions,
		Truncated:   truncated,
	}, nil
}

// listPermissionPairs resolves which (type, relation) pairs to enumerate. With
// both filters present no model read is needed; otherwise the active model's
// type/relation map is filtered down, sorted for deterministic output.
func (g *graphqlProvider) listPermissionPairs(ctx context.Context, relationFilter, typeFilter string) ([]typeRelation, error) {
	if relationFilter != "" && typeFilter != "" {
		return []typeRelation{{objType: typeFilter, relation: relationFilter}}, nil
	}
	typeRels, err := g.AuthzEngine.TypeRelations(ctx)
	if err != nil {
		return nil, err
	}
	pairs := make([]typeRelation, 0)
	for objType, relations := range typeRels {
		if typeFilter != "" && objType != typeFilter {
			continue
		}
		for _, relation := range relations {
			if relationFilter != "" && relation != relationFilter {
				continue
			}
			pairs = append(pairs, typeRelation{objType: objType, relation: relation})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].objType != pairs[j].objType {
			return pairs[i].objType < pairs[j].objType
		}
		return pairs[i].relation < pairs[j].relation
	})
	return pairs, nil
}
