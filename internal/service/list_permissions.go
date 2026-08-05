package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// maxConcurrentFgaListCalls bounds the parallel ListObjects expansions issued
// by an unfiltered ListPermissions call so one request cannot saturate the
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
// an empty input returns ALL permissions the subject holds. Transport-agnostic
// port of the former graphqlProvider.ListPermissions.
//
// SUBJECT TRUST GATE: same rules as CheckPermissions (token subject by
// default; explicit `user` for super-admins or self). The result set is
// capped at maxFgaListResults and `truncated` reports when the cap was hit:
// listing is an expensive enumeration surface.
// Permission: authorized user.
func (p *provider) ListPermissions(ctx context.Context, meta RequestMetadata, params *model.ListPermissionsInput) (*model.ListPermissionsResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ListPermissions").Logger()
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	if params == nil {
		params = &model.ListPermissionsInput{}
	}
	relationFilter := strings.TrimSpace(refs.StringValue(params.Relation))
	typeFilter := strings.TrimSpace(refs.StringValue(params.ObjectType))
	subject, err := p.resolveFgaSubject(ctx, meta, refs.StringValue(params.User))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve subject")
		return nil, nil, err
	}

	start := time.Now()
	pairs, err := p.listPermissionPairs(ctx, relationFilter, typeFilter)
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpListPermissions, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to resolve model type relations; denying")
		return nil, nil, PermissionDenied("authorization list failed")
	}

	// For an ordinary caller this is exactly [subject]. For an RFC 8693
	// delegated caller it is [agent:<client_id>, user:<sub>], and the answer is
	// the INTERSECTION of what each can reach — the same rule CheckPermissions
	// applies, expressed over enumerated object sets instead of a yes/no.
	//
	// Enumeration must intersect too: without it an agent could not ACT on an
	// object (CheckPermissions denies) yet would still see it listed, which
	// leaks the delegating user's resource names to an agent that was never
	// granted them.
	//
	// An explicitly supplied `user` (super-admin only) is never intersected —
	// the caller is asking about that subject, not acting as it.
	subjects := []string{subject}
	if strings.TrimSpace(refs.StringValue(params.User)) == "" {
		if resolved := p.delegationSubjects(ctx, subject); len(resolved) > 0 {
			subjects = resolved
		}
	}

	// Enumerate each (subject, pair) with bounded concurrency; results stay
	// positionally aligned so aggregation order is deterministic.
	perSubject := make([][][]string, len(subjects))
	for s := range subjects {
		perSubject[s] = make([][]string, len(pairs))
	}
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(maxConcurrentFgaListCalls)
	for s, subj := range subjects {
		for i, pair := range pairs {
			eg.Go(func() error {
				objects, lerr := p.AuthzEngine.ListObjects(egCtx, subj, pair.relation, pair.objType)
				if lerr != nil {
					return lerr
				}
				perSubject[s][i] = objects
				return nil
			})
		}
	}
	egErr := eg.Wait()

	// Fold to one object set per pair. With a single subject this is a copy and
	// the outcome is identical to the pre-delegation behaviour.
	results := make([][]string, len(pairs))
	if egErr == nil {
		for i := range pairs {
			results[i] = perSubject[0][i]
			for s := 1; s < len(subjects); s++ {
				results[i] = intersectObjects(results[i], perSubject[s][i])
			}
		}
	}
	metrics.ObserveFgaCheckDuration(metrics.FgaOpListPermissions, time.Since(start).Seconds())
	if egErr != nil {
		metrics.RecordFgaOperation(metrics.FgaOpListPermissions, metrics.FgaResultError)
		log.Debug().Err(egErr).Msg("ListPermissions failed; denying")
		return nil, nil, PermissionDenied("authorization list failed")
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
	}, nil, nil
}

// listPermissionPairs resolves which (type, relation) pairs to enumerate. With
// both filters present no model read is needed; otherwise the active model's
// type/relation map is filtered down, sorted for deterministic output.
func (p *provider) listPermissionPairs(ctx context.Context, relationFilter, typeFilter string) ([]typeRelation, error) {
	if relationFilter != "" && typeFilter != "" {
		return []typeRelation{{objType: typeFilter, relation: relationFilter}}, nil
	}
	typeRels, err := p.AuthzEngine.TypeRelations(ctx)
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

// intersectObjects returns the objects present in both slices, preserving the
// order of a so enumeration stays deterministic.
//
// Used to fold a delegated caller's per-subject enumerations into the set the
// agent AND the delegating user can both reach.
func intersectObjects(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	inB := make(map[string]struct{}, len(b))
	for _, o := range b {
		inB[o] = struct{}{}
	}
	out := make([]string, 0, len(a))
	for _, o := range a {
		if _, ok := inB[o]; ok {
			out = append(out, o)
		}
	}
	return out
}
