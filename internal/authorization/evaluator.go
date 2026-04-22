package authorization

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// MaxPrincipalPermissionEvaluations caps GetPrincipalPermissions at this many
// resource*scope evaluations per call. Callers hitting the cap receive a sentinel
// error so they can detect incomplete output.
const MaxPrincipalPermissionEvaluations = 10000

// policyResult holds the outcome of a single policy evaluation.
type policyResult struct {
	granted    bool
	policyName string
}

// CheckPermission evaluates whether a principal can perform a scope on a resource.
// It follows this sequence:
//  1. Validate inputs
//  2. Check MaxScopes ceiling
//  3. Check cache
//  4. Query storage for matching permissions
//  5. Evaluate policies using decision strategies
//  6. Cache and return result
//
// Every terminal path records exactly one metrics.RecordAuthzCheck call, and
// AuthzCheckDuration is observed via defer.
func (p *provider) CheckPermission(ctx context.Context, principal *Principal, resource string, scope string) (result *CheckResult, err error) {
	start := time.Now()
	defer func() {
		metrics.AuthzCheckDuration.Observe(time.Since(start).Seconds())
	}()

	mode := p.config.Enforcement

	// Validate inputs.
	if principal == nil {
		metrics.RecordAuthzCheck(mode, metrics.AuthzResultError)
		return nil, fmt.Errorf("principal is required")
	}
	if !isValidIdentifier(principal.ID) {
		metrics.RecordAuthzCheck(mode, metrics.AuthzResultError)
		return nil, fmt.Errorf("invalid principal ID: %q", principal.ID)
	}
	if !isValidIdentifier(resource) {
		metrics.RecordAuthzCheck(mode, metrics.AuthzResultError)
		return nil, fmt.Errorf("invalid resource: %q", resource)
	}
	if !isValidIdentifier(scope) {
		metrics.RecordAuthzCheck(mode, metrics.AuthzResultError)
		return nil, fmt.Errorf("invalid scope: %q", scope)
	}

	// MaxScopes ceiling.
	if principal.MaxScopes != nil {
		scopeStr := resource + ":" + scope
		found := false
		for _, ms := range principal.MaxScopes {
			if ms == scopeStr {
				found = true
				break
			}
		}
		if !found {
			p.log.Debug().
				Str("principal_id", principal.ID).
				Str("resource", resource).
				Str("scope", scope).
				Msg("denied by MaxScopes ceiling")
			metrics.RecordAuthzCheck(mode, metrics.AuthzResultDenied)
			return &CheckResult{Allowed: false}, nil
		}
	}

	// Cache.
	cacheKey := evalKey(principal.ID, resource, scope)
	if cached, ok := p.cache.get(cacheKey); ok {
		allowed := cached == "true"
		p.log.Debug().
			Str("principal_id", principal.ID).
			Str("resource", resource).
			Str("scope", scope).
			Bool("allowed", allowed).
			Msg("authorization cache hit")
		if allowed {
			metrics.RecordAuthzCheck(mode, metrics.AuthzResultAllowed)
		} else {
			metrics.RecordAuthzCheck(mode, metrics.AuthzResultDenied)
		}
		return &CheckResult{Allowed: allowed}, nil
	}

	// Resource/scope existence. Fail-closed: if the probe itself errors, surface
	// the error to the caller rather than falling through to handleNoPermission
	// (which in permissive mode would incorrectly allow a DB-blip'd request).
	knownResource, err := p.validateResourceExists(ctx, resource)
	if err != nil {
		metrics.RecordAuthzCheck(mode, metrics.AuthzResultError)
		return nil, err
	}
	knownScope := true
	if knownResource {
		// Only probe scope when resource is valid; avoids a second lookup on
		// the unknown-resource path.
		knownScope, err = p.validateScopeExists(ctx, scope)
		if err != nil {
			metrics.RecordAuthzCheck(mode, metrics.AuthzResultError)
			return nil, err
		}
	}
	if !knownResource || !knownScope {
		// Unknown identifier — skip counter bumps (DoS guard for attacker-
		// controlled inputs from the public REST endpoint).
		return p.handleNoPermission(mode, cacheKey, principal, resource, scope, false /* isKnown */), nil
	}

	// Permissions.
	perms, err := p.storageProvider.GetPermissionsForResourceScope(ctx, resource, scope)
	if err != nil {
		metrics.RecordAuthzCheck(mode, metrics.AuthzResultError)
		return nil, fmt.Errorf("failed to query permissions: %w", err)
	}
	if len(perms) == 0 {
		// Known (resource, scope) but no permission row — this is the signal
		// we DO want to track for rollout.
		return p.handleNoPermission(mode, cacheKey, principal, resource, scope, true /* isKnown */), nil
	}

	// Policy evaluation.
	for _, perm := range perms {
		allowed, matchedPolicy := p.evaluatePermission(principal, perm)
		if allowed {
			p.cache.set(cacheKey, "true")
			p.log.Debug().
				Str("principal_id", principal.ID).
				Str("resource", resource).
				Str("scope", scope).
				Str("matched_policy", matchedPolicy).
				Msg("authorization granted")
			metrics.RecordAuthzCheck(mode, metrics.AuthzResultAllowed)
			return &CheckResult{Allowed: true, MatchedPolicy: matchedPolicy}, nil
		}
	}

	// No permission granted access.
	p.cache.set(cacheKey, "false")
	p.log.Debug().
		Str("principal_id", principal.ID).
		Str("resource", resource).
		Str("scope", scope).
		Msg("authorization denied: no matching policy")
	metrics.RecordAuthzCheck(mode, metrics.AuthzResultDenied)
	return &CheckResult{Allowed: false}, nil
}

// handleNoPermission returns a deny or allow result based on enforcement mode.
// In permissive mode, it emits a rate-limited warn log (one line per
// (resource,scope) per window) and allows; in enforcing mode, it denies.
// The result is cached (negative caching) and the checks_total metric is bumped.
//
// The isKnown parameter reports whether (resource, scope) are both registered
// in the DB. Counters and the warn-limiter are only bumped for known pairs to
// prevent unbounded growth of cache.counters / warnLimiter.last from attacker-
// controlled input on the public /api/v1/check-permission REST endpoint.
func (p *provider) handleNoPermission(mode, cacheKey string, principal *Principal, resource, scope string, isKnown bool) *CheckResult {
	if isKnown {
		// Only track rollout signal for registered (resource, scope) pairs.
		// Unknown identifiers are rejected here to prevent unbounded counter
		// growth from attacker-controlled input on the public REST endpoint.
		p.cache.bumpUnmatched(resource, scope)
		metrics.RecordAuthzUnmatched(mode)
	}

	if mode == constants.AuthorizationEnforcementPermissive {
		if isKnown && p.warnLimiter.allow(resource+":"+scope) {
			p.log.Warn().
				Bool("authz.unmatched", true).
				Str("mode", mode).
				Str("principal_id", principal.ID).
				Str("resource", resource).
				Str("scope", scope).
				Msg("no matching permission (permissive: allowing)")
		}
		p.cache.set(cacheKey, "true")
		metrics.RecordAuthzCheck(mode, metrics.AuthzResultUnmatchedAllowed)
		return &CheckResult{Allowed: true}
	}

	p.cache.set(cacheKey, "false")
	metrics.RecordAuthzCheck(mode, metrics.AuthzResultUnmatchedDenied)
	return &CheckResult{Allowed: false}
}

// evaluatePermission evaluates all policies attached to a single permission
// and combines their results using the permission's decision strategy.
func (p *provider) evaluatePermission(principal *Principal, perm *schemas.PermissionWithPolicies) (bool, string) {
	if len(perm.Policies) == 0 {
		return false, ""
	}

	results := make([]policyResult, 0, len(perm.Policies))
	for i := range perm.Policies {
		policy := &perm.Policies[i]
		granted := p.evaluatePolicy(principal, policy)
		results = append(results, policyResult{
			granted:    granted,
			policyName: policy.PolicyName,
		})
	}

	return resolveDecision(results, perm.DecisionStrategy)
}

// evaluatePolicy evaluates a single policy against the principal.
// It checks whether the principal matches any of the policy's targets,
// then applies the policy's logic (positive = grant, negative = deny).
func (p *provider) evaluatePolicy(principal *Principal, policy *schemas.PolicyWithTargets) bool {
	if len(policy.Targets) == 0 {
		return false
	}

	var matched bool
	switch policy.Type {
	case constants.PolicyTypeRole:
		matched = evaluateRoleTargets(policy.Targets, principal.Roles, policy.DecisionStrategy)
	case constants.PolicyTypeUser:
		matched = evaluateUserTargets(policy.Targets, principal.ID)
	default:
		// Unknown policy type -- fail closed.
		p.log.Warn().
			Str("policy_type", policy.Type).
			Str("policy_name", policy.PolicyName).
			Msg("unknown policy type, denying")
		return false
	}

	// Apply logic: positive policies grant on match, negative policies deny on match.
	if policy.Logic == constants.PolicyLogicNegative {
		return !matched
	}
	return matched
}

// evaluateRoleTargets checks whether any (affirmative) or all (unanimous) of the
// role targets match the principal's roles.
func evaluateRoleTargets(targets []schemas.PolicyTargetView, roles []string, strategy string) bool {
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	switch strategy {
	case constants.DecisionStrategyUnanimous:
		// All role targets must be present in the principal's roles.
		evaluated := false
		for _, t := range targets {
			if t.TargetType != constants.TargetTypeRole {
				continue
			}
			evaluated = true
			if _, ok := roleSet[t.TargetValue]; !ok {
				return false
			}
		}
		return evaluated // false if no role targets existed
	default:
		// Affirmative (default): any role target match is sufficient.
		for _, t := range targets {
			if t.TargetType != constants.TargetTypeRole {
				continue
			}
			if _, ok := roleSet[t.TargetValue]; ok {
				return true
			}
		}
		return false
	}
}

// evaluateUserTargets checks whether any of the user targets match the principal's ID.
func evaluateUserTargets(targets []schemas.PolicyTargetView, principalID string) bool {
	for _, t := range targets {
		if t.TargetType == constants.TargetTypeUser && t.TargetValue == principalID {
			return true
		}
	}
	return false
}

// resolveDecision combines multiple policy results using the given strategy.
// Affirmative (default): any grant wins.
// Unanimous: all must grant.
func resolveDecision(results []policyResult, strategy string) (bool, string) {
	if len(results) == 0 {
		return false, ""
	}

	switch strategy {
	case constants.DecisionStrategyUnanimous:
		// All policies must grant.
		for _, r := range results {
			if !r.granted {
				return false, ""
			}
		}
		return true, results[0].policyName
	default:
		// Affirmative: first grant wins.
		for _, r := range results {
			if r.granted {
				return true, r.policyName
			}
		}
		return false, ""
	}
}

// GetPrincipalPermissions returns all granted resource:scope pairs for a principal.
// It iterates all known resources and scopes, checking each combination.
func (p *provider) GetPrincipalPermissions(ctx context.Context, principal *Principal) ([]ResourceScope, error) {
	if principal == nil {
		return nil, fmt.Errorf("principal is required")
	}
	if !isValidIdentifier(principal.ID) {
		return nil, fmt.Errorf("invalid principal ID: %q", principal.ID)
	}

	// Fetch all resources.
	resources, err := p.fetchAllResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Fetch all scopes.
	scopes, err := p.fetchAllScopes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list scopes: %w", err)
	}

	// Hard ceiling: refuse to enumerate O(R*S) permissions beyond the cap.
	// A tenant with, e.g., 1000 resources * 100 scopes = 100k CheckPermission
	// calls per request would otherwise saturate the authz subsystem.
	if int64(len(resources))*int64(len(scopes)) > int64(MaxPrincipalPermissionEvaluations) {
		return nil, fmt.Errorf("too many permissions to enumerate: %d resources * %d scopes exceeds cap of %d",
			len(resources), len(scopes), MaxPrincipalPermissionEvaluations)
	}

	var granted []ResourceScope
	for _, res := range resources {
		for _, sc := range scopes {
			result, err := p.CheckPermission(ctx, principal, res, sc)
			if err != nil {
				p.log.Warn().Err(err).
					Str("resource", res).
					Str("scope", sc).
					Msg("error checking permission, skipping")
				continue
			}
			if result.Allowed {
				granted = append(granted, ResourceScope{
					Resource: res,
					Scope:    sc,
				})
			}
		}
	}

	return granted, nil
}

// InvalidateCache removes cached authorization data matching the given prefix.
func (p *provider) InvalidateCache(ctx context.Context, prefix string) error {
	p.cache.deleteByPrefix(prefix)
	p.log.Debug().Str("prefix", prefix).Msg("authorization cache invalidated")
	return nil
}

// validateResourceExists reports whether the given resource is registered.
// Returns (true, nil) if known; (false, nil) if definitively unknown;
// (false, err) if the storage probe itself failed.
//
// A probe error must NOT be masked as "unknown" — previously this helper
// returned nil on DB failure, which in permissive mode caused the caller to
// fall through to allow. We now fail-closed on probe error so a transient DB
// blip cannot flip a legitimate unknown-resource path to Allowed:true.
func (p *provider) validateResourceExists(ctx context.Context, resource string) (bool, error) {
	cacheKey := validResourcesKey()
	if set, ok := p.cache.getValidSet(cacheKey); ok {
		_, found := set[resource]
		return found, nil
	}

	names, err := p.fetchAllResources(ctx)
	if err != nil {
		return false, fmt.Errorf("probe resources: %w", err)
	}

	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	p.cache.setValidSet(cacheKey, set)

	_, found := set[resource]
	return found, nil
}

// validateScopeExists reports whether the given scope is registered.
// Returns (true, nil) if known; (false, nil) if definitively unknown;
// (false, err) if the storage probe itself failed.
func (p *provider) validateScopeExists(ctx context.Context, scope string) (bool, error) {
	cacheKey := validScopesKey()
	if set, ok := p.cache.getValidSet(cacheKey); ok {
		_, found := set[scope]
		return found, nil
	}

	names, err := p.fetchAllScopes(ctx)
	if err != nil {
		return false, fmt.Errorf("probe scopes: %w", err)
	}

	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	p.cache.setValidSet(cacheKey, set)

	_, found := set[scope]
	return found, nil
}

// fetchAllResources retrieves all resource names from storage using pagination.
func (p *provider) fetchAllResources(ctx context.Context) ([]string, error) {
	var names []string
	page := int64(1)
	limit := int64(100)

	for {
		pagination := &model.Pagination{
			Limit:  limit,
			Offset: (page - 1) * limit,
			Page:   page,
		}
		resources, paginationResult, err := p.storageProvider.ListResources(ctx, pagination)
		if err != nil {
			return nil, err
		}
		for _, r := range resources {
			names = append(names, r.Name)
		}
		// If we got fewer results than the limit, or reached the total, we're done.
		if int64(len(resources)) < limit || (paginationResult != nil && paginationResult.Total <= page*limit) {
			break
		}
		page++
	}

	return names, nil
}

// fetchAllScopes retrieves all scope names from storage using pagination.
func (p *provider) fetchAllScopes(ctx context.Context) ([]string, error) {
	var names []string
	page := int64(1)
	limit := int64(100)

	for {
		pagination := &model.Pagination{
			Limit:  limit,
			Offset: (page - 1) * limit,
			Page:   page,
		}
		scopes, paginationResult, err := p.storageProvider.ListScopes(ctx, pagination)
		if err != nil {
			return nil, err
		}
		for _, s := range scopes {
			names = append(names, s.Name)
		}
		if int64(len(scopes)) < limit || (paginationResult != nil && paginationResult.Total <= page*limit) {
			break
		}
		page++
	}

	return names, nil
}
