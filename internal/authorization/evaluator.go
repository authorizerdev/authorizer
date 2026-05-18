package authorization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
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

// Boolean-valued cache entries use these sentinel strings. Any other value
// stored under an evalKey is a bug — the lookup branch below treats it as a
// cache miss (returns to the full evaluation path) rather than silently
// coercing to false.
const (
	cacheValTrue  = "true"
	cacheValFalse = "false"
)

// policyResult holds the outcome of a single policy evaluation.
type policyResult struct {
	denied     bool
	granted    bool
	policyName string
}

// CheckPermission evaluates whether a principal can perform a scope on a resource.
// It is fail-closed: any missing permission row or unknown (resource, scope) pair
// results in a deny. It follows this sequence:
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

	// Validate inputs.
	if principal == nil {
		metrics.RecordAuthzCheck(metrics.AuthzResultError)
		return nil, fmt.Errorf("principal is required")
	}
	if !isValidIdentifier(principal.ID) {
		metrics.RecordAuthzCheck(metrics.AuthzResultError)
		return nil, fmt.Errorf("invalid principal ID: %q", principal.ID)
	}
	if !isValidIdentifier(resource) {
		metrics.RecordAuthzCheck(metrics.AuthzResultError)
		return nil, fmt.Errorf("invalid resource: %q", resource)
	}
	if !isValidIdentifier(scope) {
		metrics.RecordAuthzCheck(metrics.AuthzResultError)
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
			metrics.RecordAuthzCheck(metrics.AuthzResultDenied)
			return &CheckResult{Allowed: false}, nil
		}
	}

	// Cache.
	cacheKey := evalKey(principal, resource, scope)
	if cached, ok := p.cache.get(cacheKey); ok {
		switch cached {
		case cacheValTrue:
			p.log.Debug().
				Str("principal_id", principal.ID).
				Str("resource", resource).
				Str("scope", scope).
				Bool("allowed", true).
				Msg("authorization cache hit")
			metrics.RecordAuthzCheck(metrics.AuthzResultAllowed)
			return &CheckResult{Allowed: true}, nil
		case cacheValFalse:
			p.log.Debug().
				Str("principal_id", principal.ID).
				Str("resource", resource).
				Str("scope", scope).
				Bool("allowed", false).
				Msg("authorization cache hit")
			metrics.RecordAuthzCheck(metrics.AuthzResultDenied)
			return &CheckResult{Allowed: false}, nil
		default:
			// Unexpected cache value — treat as a miss and fall through to full eval.
			p.log.Warn().Str("cache_key", cacheKey).Str("value", cached).
				Msg("authz: unexpected cached eval value, ignoring")
		}
	}

	// Resource/scope existence. Fail-closed: if the probe itself errors, surface the error to
	// the caller rather than falling through to handleNoPermission.
	knownResource, err := p.validateResourceExists(ctx, resource)
	if err != nil {
		metrics.RecordAuthzCheck(metrics.AuthzResultError)
		return nil, err
	}
	knownScope := true
	if knownResource {
		// Only probe scope when resource is valid; avoids a second lookup on
		// the unknown-resource path.
		knownScope, err = p.validateScopeExists(ctx, scope)
		if err != nil {
			metrics.RecordAuthzCheck(metrics.AuthzResultError)
			return nil, err
		}
	}
	if !knownResource || !knownScope {
		// Unknown identifier — skip counter bumps (DoS guard for attacker-
		// controlled inputs reaching CheckPermission, e.g. via GraphQL
		// myPermissions / required_permissions on authenticated endpoints).
		return p.handleNoPermission(cacheKey, principal, resource, scope, false /* isKnown */), nil
	}

	// Permissions.
	perms, err := p.storageProvider.GetPermissionsForResourceScope(ctx, resource, scope)
	if err != nil {
		metrics.RecordAuthzCheck(metrics.AuthzResultError)
		return nil, fmt.Errorf("failed to query permissions: %w", err)
	}
	if len(perms) == 0 {
		// Known (resource, scope) but no permission row — this is the signal
		// we DO want to track for rollout.
		return p.handleNoPermission(cacheKey, principal, resource, scope, true /* isKnown */), nil
	}

	// Policy evaluation. Track the first non-empty deny attribution so we
	// can surface it on the deny path for audit/debugging — resolveDecision
	// returns the denying policy name when an explicit deny fires; empty
	// strings mean "no policy contributed a verdict."
	var denyMatchedPolicy string
	for _, perm := range perms {
		allowed, matchedPolicy := p.evaluatePermission(principal, perm)
		if allowed {
			p.cache.set(cacheKey, cacheValTrue)
			p.log.Debug().
				Str("principal_id", principal.ID).
				Str("resource", resource).
				Str("scope", scope).
				Str("matched_policy", matchedPolicy).
				Msg("authorization granted")
			metrics.RecordAuthzCheck(metrics.AuthzResultAllowed)
			return &CheckResult{Allowed: true, MatchedPolicy: matchedPolicy}, nil
		}
		if denyMatchedPolicy == "" && matchedPolicy != "" {
			denyMatchedPolicy = matchedPolicy
		}
	}

	// No permission granted access.
	p.cache.set(cacheKey, cacheValFalse)
	p.log.Debug().
		Str("principal_id", principal.ID).
		Str("resource", resource).
		Str("scope", scope).
		Str("matched_policy", denyMatchedPolicy).
		Msg("authorization denied")
	metrics.RecordAuthzCheck(metrics.AuthzResultDenied)
	return &CheckResult{Allowed: false, MatchedPolicy: denyMatchedPolicy}, nil
}

// handleNoPermission returns a deny result for an unmatched (resource, scope)
// pair. The isKnown parameter reports whether the pair is both registered in
// the DB — counters are bumped only for known pairs to prevent unbounded
// growth of cache.counters from attacker-controlled input reaching
// CheckPermission via authenticated GraphQL (myPermissions /
// required_permissions).
func (p *provider) handleNoPermission(cacheKey string, _ *Principal, resource, scope string, isKnown bool) *CheckResult {
	if isKnown {
		p.cache.bumpUnmatched(resource, scope)
		metrics.RecordAuthzUnmatched()
	}
	p.cache.set(cacheKey, cacheValFalse)
	metrics.RecordAuthzCheck(metrics.AuthzResultUnmatched)
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
		denied, granted := p.evaluatePolicy(principal, policy)
		results = append(results, policyResult{
			denied:     denied,
			granted:    granted,
			policyName: policy.PolicyName,
		})
	}

	return resolveDecision(results, perm.DecisionStrategy)
}

// evaluatePolicy evaluates a single policy against the principal.
// It checks whether the principal matches any of the policy's targets,
// then applies the policy's logic (positive = grant, negative = deny).
func (p *provider) evaluatePolicy(principal *Principal, policy *schemas.PolicyWithTargets) (denied bool, granted bool) {
	if len(policy.Targets) == 0 {
		return false, false
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
		return true, false
	}

	// Apply logic: positive policies grant on match, negative policies deny on match.
	if policy.Logic == constants.PolicyLogicNegative {
		return matched, false
	}
	return false, matched
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
// Any explicit deny wins. Otherwise, affirmative grants on any allow, while
// unanimous requires every policy to grant.
func resolveDecision(results []policyResult, strategy string) (bool, string) {
	if len(results) == 0 {
		return false, ""
	}

	for _, r := range results {
		if r.denied {
			return false, r.policyName
		}
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

// evalKey constructs a cache key for an authorization evaluation result. The
// effective roles and delegation ceiling are part of the key because the same
// principal ID can legitimately evaluate to different answers across sessions.
func evalKey(principal *Principal, resource, scope string) string {
	fp := principalFingerprint(principal)
	return fmt.Sprintf("authz:eval:%s:%s:%s:%s", principal.ID, fp, resource, scope)
}

func principalFingerprint(principal *Principal) string {
	roles := append([]string(nil), principal.Roles...)
	sort.Strings(roles)
	maxScopes := append([]string(nil), principal.MaxScopes...)
	sort.Strings(maxScopes)

	h := sha256.New()
	_, _ = h.Write([]byte(principal.Type))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.Join(roles, "\x00")))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.Join(maxScopes, "\x00")))
	return hex.EncodeToString(h.Sum(nil))[:16]
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
// Called by admin mutations when permissions/policies change.
func (p *provider) InvalidateCache(ctx context.Context, prefix string) {
	p.cache.deleteByPrefix(prefix)
	p.log.Debug().Str("prefix", prefix).Msg("authorization cache invalidated")
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
