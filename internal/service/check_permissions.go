package service

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
// check, in order. A single check is simply a list of one. Transport-agnostic
// port of the former graphqlProvider.CheckPermissions.
//
// SUBJECT TRUST GATE: the subject defaults to the authenticated caller's token
// subject; an explicit `user` is honored only for super-admins or when it
// equals the caller's own subject (see resolveFgaSubject). Fail-closed: any
// engine error denies.
// Permission: authorized user.
func (p *provider) CheckPermissions(ctx context.Context, meta RequestMetadata, params *model.CheckPermissionsInput) (*model.CheckPermissionsResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "CheckPermissions").Logger()
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	if params == nil || len(params.Checks) == 0 {
		return nil, nil, InvalidArgument("at least one check is required")
	}
	if len(params.Checks) > maxPermissionChecks {
		return nil, nil, InvalidArgument(fmt.Sprintf("too many checks: max %d per request", maxPermissionChecks))
	}
	// Resolved once and threaded through both gates below; see fgaCaller.
	caller, err := p.resolveFgaCaller(ctx, meta)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, nil, err
	}
	subject, err := p.resolveFgaSubject(ctx, meta, caller, refs.StringValue(params.User))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve subject")
		return nil, nil, err
	}
	reqContext, err := p.checkRequestContext(ctx, meta, caller, params.Context)
	if err != nil {
		log.Debug().Err(err).Msg("Rejected request context")
		return nil, nil, err
	}
	// Contextual tuples are caller-asserted relationships: attaching one that
	// grants the caller the checked relation yields allowed=true. For an agent
	// (delegated token) that turns a "check before acting" guardrail into one
	// it can talk itself past, so delegated callers may not send them.
	if caller.isDelegated() {
		for _, c := range params.Checks {
			if c != nil && len(c.ContextualTuples) > 0 {
				return nil, nil, PermissionDenied("a delegated token may not supply contextual tuples")
			}
		}
	}
	// For an ordinary caller this is exactly [subject] and everything below is
	// unchanged. For an RFC 8693 delegated token it is
	// [agent:<client_id>, user:<sub>], and EVERY subject must be allowed —
	// effective authority is perms(agent) ∩ perms(user). See delegationSubjects.
	//
	// This is deliberately NOT conditioned on `user` being absent. It used to
	// be, and that made the intersection defeatable with one parameter: a
	// delegated agent echoing back its own subject — which resolveFgaSubject
	// accepts, since it is exactly what the token proves — skipped the
	// expansion and was authorized as the user alone. The gate belongs on WHO
	// the caller is, never on what they typed. A super-admin naming a different
	// subject is not delegated, so this is the identity operation for them.
	subjects, err := p.delegationSubjects(ctx, caller, subject, metrics.FgaOpCheckPermissions)
	if err != nil {
		metrics.RecordFgaCheck(metrics.FgaOpCheckPermissions, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to resolve delegation subjects; denying")
		return nil, nil, err
	}

	// Requests are laid out subject-major: all checks for subject[0], then all
	// for subject[1]. Result i for check j therefore lives at
	// index i*len(checks)+j, which is how the intersection is folded below.
	requests := make([]engine.CheckRequest, 0, len(params.Checks)*len(subjects))
	for _, s := range subjects {
		for _, c := range params.Checks {
			if c == nil || strings.TrimSpace(c.Relation) == "" || strings.TrimSpace(c.Object) == "" {
				return nil, nil, InvalidArgument("each check requires relation and object")
			}
			ctxTuples, err := toContextualTuples(c.ContextualTuples)
			if err != nil {
				return nil, nil, err
			}
			requests = append(requests, engine.CheckRequest{
				User:             s,
				Relation:         c.Relation,
				Object:           c.Object,
				ContextualTuples: ctxTuples,
				Context:          reqContext,
			})
		}
	}
	start := time.Now()
	results, err := p.AuthzEngine.BatchCheck(ctx, requests)
	metrics.ObserveFgaCheckDuration(metrics.FgaOpCheckPermissions, time.Since(start).Seconds())
	if err != nil {
		// Fail closed for the whole call.
		metrics.RecordFgaCheck(metrics.FgaOpCheckPermissions, metrics.FgaResultError)
		log.Debug().Err(err).Msg("CheckPermissions failed; denying")
		return nil, nil, PermissionDenied("authorization check failed")
	}
	if len(results) != len(requests) {
		// Fail closed rather than mis-index the fold below.
		metrics.RecordFgaCheck(metrics.FgaOpCheckPermissions, metrics.FgaResultError)
		log.Debug().Int("want", len(requests)).Int("got", len(results)).
			Msg("CheckPermissions: engine returned an unexpected result count; denying")
		return nil, nil, PermissionDenied("authorization check failed")
	}

	// Fold the subject-major results into one decision per check by AND-ing
	// across subjects. With a single subject this is the identity operation and
	// the outcome is bit-for-bit what it was before delegation existed.
	n := len(params.Checks)
	// subjects[0] is the agent only when delegationSubjects expanded the list;
	// with one subject there is nothing to attribute a denial to.
	delegated := len(subjects) > 1
	out := &model.CheckPermissionsResponse{Results: make([]*model.PermissionCheckResult, 0, n)}
	for j := 0; j < n; j++ {
		allowed := true
		deniedBy := -1
		for i := range subjects {
			if !results[i*n+j].Allowed {
				allowed = false
				deniedBy = i
				break
			}
		}
		if delegated {
			// Attribute the denial so an operator can tell "grant the agent a
			// tuple" from "the user genuinely lacks access" — see
			// metrics.FgaDelegatedDeniedByAgent.
			switch {
			case allowed:
				metrics.RecordFgaDelegatedCheck(metrics.FgaOpCheckPermissions, metrics.FgaDelegatedAllowed)
			case deniedBy == 0:
				metrics.RecordFgaDelegatedCheck(metrics.FgaOpCheckPermissions, metrics.FgaDelegatedDeniedByAgent)
			default:
				metrics.RecordFgaDelegatedCheck(metrics.FgaOpCheckPermissions, metrics.FgaDelegatedDeniedByUser)
			}
		}
		// Record each decision so adoption/denial rates reflect every pair.
		metrics.RecordFgaCheckResult(metrics.FgaOpCheckPermissions, allowed)
		out.Results = append(out.Results, &model.PermissionCheckResult{
			Relation: params.Checks[j].Relation,
			Object:   params.Checks[j].Object,
			Allowed:  allowed,
		})
	}
	return out, nil, nil
}

// maxCheckContextParams caps the request-time condition parameters on one
// CheckPermissions call.
const maxCheckContextParams = 32

// checkRequestContext validates caller-supplied condition context.
//
// Request context feeds condition evaluation, so whoever supplies it can make
// a condition pass: a caller asserting its own parameters could defeat any
// condition that reads them. It is therefore accepted only from a super-admin
// who is not acting through a delegated token, and never for
// engine.ContextKeyCurrentTime, which the engine always sets from the server
// clock. Anything else is rejected rather than silently dropped.
func (p *provider) checkRequestContext(ctx context.Context, meta RequestMetadata, caller fgaCaller, in map[string]any) (map[string]any, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if caller.isDelegated() {
		return nil, PermissionDenied("a delegated token may not supply condition context")
	}
	// Same positive path as requireSuperAdmin: the gRPC principal, or the admin
	// cookie/secret on the HTTP transports. Never derived from client input.
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, PermissionDenied("condition context may only be supplied by a super-admin")
	}
	if len(in) > maxCheckContextParams {
		return nil, InvalidArgument(fmt.Sprintf("too many context parameters: max %d", maxCheckContextParams))
	}
	if _, ok := in[engine.ContextKeyCurrentTime]; ok {
		return nil, InvalidArgument("current_time is supplied by the server and cannot be set in context")
	}
	return in, nil
}
