package authorization

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// policyResult holds the outcome of a single policy evaluation.
type policyResult struct {
	granted    bool
	policyName string
}

// CheckPermission evaluates whether a principal can perform a scope on a resource.
// It follows this sequence:
//  1. Check enforcement mode (disabled returns true immediately)
//  2. Validate inputs
//  3. Check MaxScopes ceiling
//  4. Check cache
//  5. Query storage for matching permissions
//  6. Evaluate policies using decision strategies
//  7. Cache and return result
func (p *provider) CheckPermission(ctx context.Context, principal *Principal, resource string, scope string) (*CheckResult, error) {
	// Step 1: Enforcement mode check.
	if p.config.Enforcement == constants.AuthorizationEnforcementDisabled {
		return &CheckResult{Allowed: true}, nil
	}

	// Step 2: Validate inputs.
	if principal == nil {
		return nil, fmt.Errorf("principal is required")
	}
	if !isValidIdentifier(principal.ID) {
		return nil, fmt.Errorf("invalid principal ID: %q", principal.ID)
	}
	if !isValidIdentifier(resource) {
		return nil, fmt.Errorf("invalid resource: %q", resource)
	}
	if !isValidIdentifier(scope) {
		return nil, fmt.Errorf("invalid scope: %q", scope)
	}

	// Step 3: MaxScopes ceiling.
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
			return &CheckResult{Allowed: false}, nil
		}
	}

	// Step 4: Check cache.
	cacheKey := evalKey(principal.ID, resource, scope)
	if cached, ok := p.cache.get(cacheKey); ok {
		allowed := cached == "true"
		p.log.Debug().
			Str("principal_id", principal.ID).
			Str("resource", resource).
			Str("scope", scope).
			Bool("allowed", allowed).
			Msg("authorization cache hit")
		return &CheckResult{Allowed: allowed}, nil
	}

	// Step 5: Check known resources and scopes.
	if err := p.validateResourceExists(ctx, resource); err != nil {
		return p.handleNoPermission(cacheKey, principal, resource, scope), nil
	}
	if err := p.validateScopeExists(ctx, scope); err != nil {
		return p.handleNoPermission(cacheKey, principal, resource, scope), nil
	}

	// Step 6: Query storage for permissions matching this resource+scope.
	perms, err := p.storageProvider.GetPermissionsForResourceScope(ctx, resource, scope)
	if err != nil {
		return nil, fmt.Errorf("failed to query permissions: %w", err)
	}

	if len(perms) == 0 {
		return p.handleNoPermission(cacheKey, principal, resource, scope), nil
	}

	// Step 7: Evaluate each permission's policies.
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
	return &CheckResult{Allowed: false}, nil
}

// handleNoPermission returns a deny or allow result based on enforcement mode.
// In permissive mode, it logs a warning and allows. In enforcing mode, it denies.
// The result is cached (negative caching).
func (p *provider) handleNoPermission(cacheKey string, principal *Principal, resource, scope string) *CheckResult {
	if p.config.Enforcement == constants.AuthorizationEnforcementPermissive {
		p.log.Warn().
			Str("principal_id", principal.ID).
			Str("resource", resource).
			Str("scope", scope).
			Msg("no matching permission found (permissive mode: allowing)")
		p.cache.set(cacheKey, "true")
		return &CheckResult{Allowed: true}
	}

	p.cache.set(cacheKey, "false")
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
		for _, t := range targets {
			if t.TargetType != constants.TargetTypeRole {
				continue
			}
			if _, ok := roleSet[t.TargetValue]; !ok {
				return false
			}
		}
		return true
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
	if p.config.Enforcement == constants.AuthorizationEnforcementDisabled {
		return nil, nil
	}

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

// validateResourceExists checks that the given resource name is registered in the DB.
// Results are cached to avoid repeated DB lookups.
func (p *provider) validateResourceExists(ctx context.Context, resource string) error {
	cacheKey := validResourcesKey()
	if cached, ok := p.cache.get(cacheKey); ok {
		if strings.Contains(","+cached+",", ","+resource+",") {
			return nil
		}
		return fmt.Errorf("unknown resource: %s", resource)
	}

	names, err := p.fetchAllResources(ctx)
	if err != nil {
		// On error, allow the request to proceed (fail open for validation,
		// the actual permission check will still be default-deny).
		return nil
	}

	p.cache.set(cacheKey, strings.Join(names, ","))

	for _, n := range names {
		if n == resource {
			return nil
		}
	}
	return fmt.Errorf("unknown resource: %s", resource)
}

// validateScopeExists checks that the given scope name is registered in the DB.
// Results are cached to avoid repeated DB lookups.
func (p *provider) validateScopeExists(ctx context.Context, scope string) error {
	cacheKey := validScopesKey()
	if cached, ok := p.cache.get(cacheKey); ok {
		if strings.Contains(","+cached+",", ","+scope+",") {
			return nil
		}
		return fmt.Errorf("unknown scope: %s", scope)
	}

	names, err := p.fetchAllScopes(ctx)
	if err != nil {
		return nil
	}

	p.cache.set(cacheKey, strings.Join(names, ","))

	for _, n := range names {
		if n == scope {
			return nil
		}
	}
	return fmt.Errorf("unknown scope: %s", scope)
}

// fetchAllResources retrieves all resource names from storage using pagination.
func (p *provider) fetchAllResources(ctx context.Context) ([]string, error) {
	var names []string
	page := int64(1)
	limit := int64(100)

	for {
		pagination := &model.Pagination{
			Limit: limit,
			Page:  page,
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
			Limit: limit,
			Page:  page,
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
