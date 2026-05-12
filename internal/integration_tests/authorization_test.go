package integration_tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/cmd"
	"github.com/authorizerdev/authorizer/internal/authorization"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthorizationCRUD tests the fine-grained authorization CRUD operations
// and permission checking.
func TestAuthorizationCRUD(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Set admin auth cookie for admin operations
	adminHash, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	// IDs collected across subtests
	var resourceID string
	var scopeID string
	var policyID string
	var permissionID string

	t.Run("should add resource", func(t *testing.T) {
		res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{
			Name:        "documents",
			Description: refs.NewStringRef("Document resource for testing"),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.ID)
		assert.Equal(t, "documents", res.Name)
		assert.NotNil(t, res.Description)
		assert.Equal(t, "Document resource for testing", *res.Description)
		resourceID = res.ID
	})

	t.Run("should add scope", func(t *testing.T) {
		res, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{
			Name:        "read",
			Description: refs.NewStringRef("Read access scope"),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.ID)
		assert.Equal(t, "read", res.Name)
		scopeID = res.ID
	})

	t.Run("should add policy", func(t *testing.T) {
		res, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
			Name:        "user-role-policy",
			Description: refs.NewStringRef("Policy for user role"),
			Type:        "role",
			Targets: []*model.PolicyTargetInput{
				{
					TargetType:  "role",
					TargetValue: "user",
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.ID)
		assert.Equal(t, "user-role-policy", res.Name)
		assert.Equal(t, "role", res.Type)
		assert.Equal(t, "positive", res.Logic)
		assert.Equal(t, "affirmative", res.DecisionStrategy)
		require.Len(t, res.Targets, 1)
		assert.Equal(t, "role", res.Targets[0].TargetType)
		assert.Equal(t, "user", res.Targets[0].TargetValue)
		policyID = res.ID
	})

	t.Run("should add permission", func(t *testing.T) {
		require.NotEmpty(t, resourceID, "resourceID must be set from prior subtest")
		require.NotEmpty(t, scopeID, "scopeID must be set from prior subtest")
		require.NotEmpty(t, policyID, "policyID must be set from prior subtest")

		res, err := ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
			Name:       "documents-read",
			ResourceID: resourceID,
			ScopeIds:   []string{scopeID},
			PolicyIds:  []string{policyID},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.ID)
		assert.Equal(t, "documents-read", res.Name)
		assert.Equal(t, "affirmative", res.DecisionStrategy)
		require.NotNil(t, res.Resource)
		assert.Equal(t, resourceID, res.Resource.ID)
		require.Len(t, res.Scopes, 1)
		assert.Equal(t, scopeID, res.Scopes[0].ID)
		require.Len(t, res.Policies, 1)
		assert.Equal(t, policyID, res.Policies[0].ID)
		permissionID = res.ID
	})

	t.Run("should list resources", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Resources(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.GreaterOrEqual(t, len(res.Resources), 1)
		assert.NotNil(t, res.Pagination)

		found := false
		for _, r := range res.Resources {
			if r.ID == resourceID {
				found = true
				assert.Equal(t, "documents", r.Name)
				break
			}
		}
		assert.True(t, found, "expected resource not found in list")
	})

	t.Run("should list scopes", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Scopes(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.GreaterOrEqual(t, len(res.Scopes), 1)
	})

	t.Run("should list policies", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Policies(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.GreaterOrEqual(t, len(res.Policies), 1)
	})

	t.Run("should list permissions", func(t *testing.T) {
		res, err := ts.GraphQLProvider.Permissions(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.GreaterOrEqual(t, len(res.Permissions), 1)
	})

	t.Run("should update resource", func(t *testing.T) {
		require.NotEmpty(t, resourceID)
		newName := "documents-updated"
		res, err := ts.GraphQLProvider.UpdateResource(ctx, &model.UpdateResourceInput{
			ID:          resourceID,
			Name:        &newName,
			Description: refs.NewStringRef("Updated description"),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, resourceID, res.ID)
		assert.Equal(t, "documents-updated", res.Name)
		assert.Equal(t, "Updated description", *res.Description)

		// Revert name for subsequent tests that reference "documents" by ID
		origName := "documents"
		_, err = ts.GraphQLProvider.UpdateResource(ctx, &model.UpdateResourceInput{
			ID:   resourceID,
			Name: &origName,
		})
		require.NoError(t, err)
	})

	t.Run("should check permission granted by role", func(t *testing.T) {
		res, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
			ID:    uuid.New().String(),
			Type:  constants.PrincipalTypeUser,
			Roles: []string{"user"},
		}, "documents", "read")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Allowed, "principal with 'user' role should have read access to documents")
	})

	t.Run("should check permission denied for wrong role", func(t *testing.T) {
		// Add an admin-only policy + a "write" scope + a permission requiring
		// the "admin" role for "write" on documents.
		adminPolicy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
			Name: "admin-only-policy",
			Type: "role",
			Targets: []*model.PolicyTargetInput{
				{
					TargetType:  "role",
					TargetValue: "admin",
				},
			},
		})
		require.NoError(t, err)

		writeScope, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{
			Name: "write",
		})
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
			Name:       "documents-write",
			ResourceID: resourceID,
			ScopeIds:   []string{writeScope.ID},
			PolicyIds:  []string{adminPolicy.ID},
		})
		require.NoError(t, err)

		res, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
			ID:    uuid.New().String(),
			Type:  constants.PrincipalTypeUser,
			Roles: []string{"user"},
		}, "documents", "write")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Allowed, "principal with 'user' role should NOT have write access requiring 'admin' role")
	})

	// Re-set admin cookie for remaining admin operations
	t.Run("should delete permission", func(t *testing.T) {
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))
		require.NotEmpty(t, permissionID)

		res, err := ts.GraphQLProvider.DeletePermission(ctx, permissionID)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Message, "deleted")
	})

	t.Run("should delete resource blocked by permission", func(t *testing.T) {
		// The "documents-write" permission still references this resource,
		// so delete should fail.
		_, err := ts.GraphQLProvider.DeleteResource(ctx, resourceID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission")
	})

	t.Run("should delete scope blocked by permission", func(t *testing.T) {
		// The "write" scope is referenced by "documents-write" permission.
		// Find the write scope ID from the scopes list.
		scopes, err := ts.GraphQLProvider.Scopes(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		var writeScopeID string
		for _, s := range scopes.Scopes {
			if s.Name == "write" {
				writeScopeID = s.ID
				break
			}
		}
		require.NotEmpty(t, writeScopeID, "write scope must exist")

		_, err = ts.GraphQLProvider.DeleteScope(ctx, writeScopeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission")
	})

	t.Run("should delete policy blocked by permission", func(t *testing.T) {
		// The "admin-only-policy" is referenced by "documents-write" permission.
		policies, err := ts.GraphQLProvider.Policies(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		var adminPolicyID string
		for _, p := range policies.Policies {
			if p.Name == "admin-only-policy" {
				adminPolicyID = p.ID
				break
			}
		}
		require.NotEmpty(t, adminPolicyID, "admin-only-policy must exist")

		_, err = ts.GraphQLProvider.DeletePolicy(ctx, adminPolicyID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission")
	})

	// Cleanup: delete the remaining permission first, then the rest
	t.Run("cleanup should delete remaining permission then resources", func(t *testing.T) {
		// Find and delete the "documents-write" permission
		perms, err := ts.GraphQLProvider.Permissions(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)

		for _, p := range perms.Permissions {
			if p.Name == "documents-write" {
				res, err := ts.GraphQLProvider.DeletePermission(ctx, p.ID)
				require.NoError(t, err)
				assert.Contains(t, res.Message, "deleted")
				break
			}
		}

		// Now resource, scope, and policy should be deletable
		res, err := ts.GraphQLProvider.DeleteResource(ctx, resourceID)
		require.NoError(t, err)
		assert.Contains(t, res.Message, "deleted")

		res, err = ts.GraphQLProvider.DeleteScope(ctx, scopeID)
		require.NoError(t, err)
		assert.Contains(t, res.Message, "deleted")

		res, err = ts.GraphQLProvider.DeletePolicy(ctx, policyID)
		require.NoError(t, err)
		assert.Contains(t, res.Message, "deleted")
	})
}

// TestCheckPermission_PermissiveDefault_NoPermissions_Allows verifies that
// permissive mode allows a check for a (resource, scope) pair that has no
// matching permission registered.
func TestCheckPermission_PermissiveDefault_NoPermissions_Allows(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementPermissive)
	_, ctx := createContext(ts)

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:   "user-1",
		Type: constants.PrincipalTypeUser,
	}, "orders", "read")

	require.NoError(t, err)
	require.True(t, result.Allowed, "permissive mode with no permissions must allow")
}

// TestCheckPermission_Enforcing_NoPermissions_Denies verifies that enforcing
// mode denies a check for a (resource, scope) pair with no matching permission.
func TestCheckPermission_Enforcing_NoPermissions_Denies(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	_, ctx := createContext(ts)

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:   "user-1",
		Type: constants.PrincipalTypeUser,
	}, "orders", "read")

	require.NoError(t, err)
	require.False(t, result.Allowed, "enforcing mode with no permissions must deny")
}

// TestCheckPermission_Permissive_WithExplicitDenyPolicy_StillDenies verifies
// that once a permission exists for the (resource, scope) and attaches a
// negative-logic policy that matches the principal, the check is denied even
// in permissive mode. Permissive only loosens the "no matching permission"
// path, not evaluated deny decisions.
func TestCheckPermission_Permissive_WithExplicitDenyPolicy_StillDenies(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementPermissive)
	req, ctx := createContext(ts)

	// Authenticate as admin for seeding operations.
	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	// Seed resource + scope + permission + negative role policy targeting "blocked-role".
	seedResourceScopePermissionWithDenyPolicy(t, ts, ctx, "orders", "read", "blocked-role")

	// Clear admin cookie — CheckPermission here is a direct provider call; no auth context needed.
	req.Header.Del("Cookie")

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:    "user-1",
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"blocked-role"},
	}, "orders", "read")

	require.NoError(t, err)
	require.False(t, result.Allowed, "explicit deny must apply even in permissive mode")
}

func TestCheckPermission_ExplicitDenyOverridesAffirmativeGrant(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	req, ctx := createContext(ts)

	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{Name: "deny-override-docs"})
	require.NoError(t, err)
	sc, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{Name: "read-deny-override"})
	require.NoError(t, err)

	positive := constants.PolicyLogicPositive
	grantPolicy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name:  "grant-user-" + uuid.New().String(),
		Type:  constants.PolicyTypeRole,
		Logic: &positive,
		Targets: []*model.PolicyTargetInput{{
			TargetType:  constants.TargetTypeRole,
			TargetValue: "user",
		}},
	})
	require.NoError(t, err)

	negative := constants.PolicyLogicNegative
	denyPolicy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name:  "deny-blocked-" + uuid.New().String(),
		Type:  constants.PolicyTypeRole,
		Logic: &negative,
		Targets: []*model.PolicyTargetInput{{
			TargetType:  constants.TargetTypeRole,
			TargetValue: "blocked-role",
		}},
	})
	require.NoError(t, err)

	_, err = ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       "deny-override-permission-" + uuid.New().String(),
		ResourceID: res.ID,
		ScopeIds:   []string{sc.ID},
		PolicyIds:  []string{grantPolicy.ID, denyPolicy.ID},
	})
	require.NoError(t, err)

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:    "user-1",
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"user", "blocked-role"},
	}, "deny-override-docs", "read-deny-override")
	require.NoError(t, err)
	require.False(t, result.Allowed, "matching negative policy must override an affirmative grant")

	result, err = ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:    "user-2",
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"user"},
	}, "deny-override-docs", "read-deny-override")
	require.NoError(t, err)
	require.True(t, result.Allowed, "non-matching negative policy must not block a positive grant")

	result, err = ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:    "user-3",
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"other-role"},
	}, "deny-override-docs", "read-deny-override")
	require.NoError(t, err)
	require.False(t, result.Allowed, "non-matching negative policy must not grant access by itself")
}

func TestCheckPermission_CacheKeyIncludesRoles(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	req, ctx := createContext(ts)

	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	seedResourceScopePermissionWithPositivePolicy(t, ts, ctx, "cached-docs", "read", "viewer")

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:    "user-1",
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"viewer"},
	}, "cached-docs", "read")
	require.NoError(t, err)
	require.True(t, result.Allowed)

	result, err = ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:   "user-1",
		Type: constants.PrincipalTypeUser,
	}, "cached-docs", "read")
	require.NoError(t, err)
	require.False(t, result.Allowed, "cached allow for viewer role must not apply to the same user without that role")
}

func TestUpdatePermission_InvalidScopeDoesNotDropExistingLinks(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	req, ctx := createContext(ts)

	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{Name: "update-safe-docs"})
	require.NoError(t, err)
	sc, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{Name: "update-safe-read"})
	require.NoError(t, err)
	policy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name: "update-safe-policy-" + uuid.New().String(),
		Type: constants.PolicyTypeRole,
		Targets: []*model.PolicyTargetInput{{
			TargetType:  constants.TargetTypeRole,
			TargetValue: "viewer",
		}},
	})
	require.NoError(t, err)
	perm, err := ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       "update-safe-permission-" + uuid.New().String(),
		ResourceID: res.ID,
		ScopeIds:   []string{sc.ID},
		PolicyIds:  []string{policy.ID},
	})
	require.NoError(t, err)

	// Capture pre-failure state. Field-level rollback is part of the contract:
	// a failed update must leave Name, Description, and DecisionStrategy
	// untouched on the persisted permission row.
	origPerm, err := ts.StorageProvider.GetPermissionByID(ctx, perm.ID)
	require.NoError(t, err)
	origName := origPerm.Name
	origDescription := origPerm.Description
	origDecision := origPerm.DecisionStrategy

	newName := "should-not-be-applied"
	newDescription := "should-not-be-applied-description"
	newDecision := constants.DecisionStrategyUnanimous
	_, err = ts.GraphQLProvider.UpdatePermission(ctx, &model.UpdatePermissionInput{
		ID:               perm.ID,
		Name:             &newName,
		Description:      &newDescription,
		DecisionStrategy: &newDecision,
		ScopeIds:         []string{"missing-scope-id"},
	})
	require.Error(t, err)

	scopes, err := ts.StorageProvider.GetPermissionScopes(ctx, perm.ID)
	require.NoError(t, err)
	require.Len(t, scopes, 1)
	require.Equal(t, sc.ID, scopes[0].ScopeID)

	// Verify field changes were rolled back. The persisted row must still hold
	// the original values; the attempted update must have written nothing.
	after, err := ts.StorageProvider.GetPermissionByID(ctx, perm.ID)
	require.NoError(t, err)
	require.Equal(t, origName, after.Name, "name must not change when update fails")
	require.Equal(t, origDescription, after.Description, "description must not change when update fails")
	require.Equal(t, origDecision, after.DecisionStrategy, "decision strategy must not change when update fails")

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:    "user-1",
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"viewer"},
	}, "update-safe-docs", "update-safe-read")
	require.NoError(t, err)
	require.True(t, result.Allowed, "failed update must not remove existing permission scope")
}

func TestAddPermission_DuplicateNameReturnsConflict(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	req, ctx := createContext(ts)

	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{Name: "duplicate-docs"})
	require.NoError(t, err)
	sc, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{Name: "duplicate-read"})
	require.NoError(t, err)
	policy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name: "duplicate-policy-" + uuid.New().String(),
		Type: constants.PolicyTypeRole,
		Targets: []*model.PolicyTargetInput{{
			TargetType:  constants.TargetTypeRole,
			TargetValue: "viewer",
		}},
	})
	require.NoError(t, err)

	input := &model.AddPermissionInput{
		Name:       "duplicate-permission",
		ResourceID: res.ID,
		ScopeIds:   []string{sc.ID},
		PolicyIds:  []string{policy.ID},
	}
	_, err = ts.GraphQLProvider.AddPermission(ctx, input)
	require.NoError(t, err)

	// The exact error wording is provider-specific (SQL emits "already exists",
	// while NoSQL backends surface their native duplicate-key errors). Only the
	// presence of an error is contractual.
	_, err = ts.GraphQLProvider.AddPermission(ctx, input)
	require.Error(t, err, "duplicate permission name must surface as an error from any storage backend")
}

// TestCheckPermission_IncrementsPrometheusCounters verifies that an unmatched
// check in permissive mode increments metrics.AuthzUnmatchedTotal by exactly
// one for the "permissive" label. The (resource, scope) pair MUST be registered
// first so that the "known but no matching permission" path is exercised —
// unknown identifiers intentionally no longer bump the counter (DoS guard).
func TestCheckPermission_IncrementsPrometheusCounters(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementPermissive)
	_, ctx := createContext(ts)

	// Seed resource + scope directly via storage (no permission). This makes
	// validateResourceExists / validateScopeExists return known=true, so the
	// subsequent CheckPermission lands on the "known, no permission" path
	// that DOES bump counters.
	seedKnownResourceScopeNoPermission(t, ts, ctx, "orders", "read")

	before := testutil.ToFloat64(metrics.AuthzUnmatchedTotal.WithLabelValues(metrics.AuthzModePermissive))

	_, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:   "user-1",
		Type: constants.PrincipalTypeUser,
	}, "orders", "read")
	require.NoError(t, err)

	after := testutil.ToFloat64(metrics.AuthzUnmatchedTotal.WithLabelValues(metrics.AuthzModePermissive))
	require.Equal(t, before+1, after, "unmatched counter must increment once per unmatched check")
}

// TestCheckPermission_UnknownResource_PermissiveStillAllows_ButDoesNotBumpUnmatchedCounter
// verifies the DoS guard: permissive mode still allows the request (so callers
// aren't broken by a typo/unknown identifier), but the unmatched counter and
// warn-limiter MUST NOT grow for attacker-controlled input. Authenticated
// callers can still reach CheckPermission with arbitrary identifiers via
// GraphQL (myPermissions / required_permissions) — without this guard they
// could flood the in-process sync.Map with arbitrary (resource, scope) pairs.
func TestCheckPermission_UnknownResource_PermissiveStillAllows_ButDoesNotBumpUnmatchedCounter(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementPermissive)
	_, ctx := createContext(ts)

	before := testutil.ToFloat64(metrics.AuthzUnmatchedTotal.WithLabelValues(metrics.AuthzModePermissive))

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID: "user-1", Type: constants.PrincipalTypeUser,
	}, "unknown-resource", "unknown-scope")

	require.NoError(t, err)
	require.True(t, result.Allowed, "permissive still allows unknown resource")

	after := testutil.ToFloat64(metrics.AuthzUnmatchedTotal.WithLabelValues(metrics.AuthzModePermissive))
	require.Equal(t, before, after, "unknown-resource calls must NOT bump the unmatched counter (DoS guard)")
}

// seedKnownResourceScopeNoPermission inserts a Resource and Scope row via the
// storage provider without attaching a Permission. This is the minimal seed
// needed to exercise the "known (resource, scope), no matching permission"
// path in CheckPermission after Fix B/C.
func seedKnownResourceScopeNoPermission(t *testing.T, ts *testSetup, _ context.Context, resource, scope string) {
	t.Helper()
	_, err := ts.StorageProvider.AddResource(context.Background(), &schemas.Resource{
		Name:        resource,
		Description: "seed (no permission) resource",
	})
	require.NoError(t, err)
	_, err = ts.StorageProvider.AddScope(context.Background(), &schemas.Scope{
		Name:        scope,
		Description: "seed (no permission) scope",
	})
	require.NoError(t, err)
}

// seedResourceScopePermissionWithDenyPolicy seeds a resource, scope, a
// negative-logic role policy targeting the given role, and a permission that
// links them. It uses the GraphQL provider (mirroring TestAuthorizationCRUD),
// so the caller must have already authenticated as admin on the request
// attached to ts.GinContext.
func seedResourceScopePermissionWithDenyPolicy(
	t *testing.T,
	ts *testSetup,
	ctx context.Context,
	resource, scope, role string,
) {
	t.Helper()

	res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{
		Name:        resource,
		Description: refs.NewStringRef("seed resource"),
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	sc, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{
		Name:        scope,
		Description: refs.NewStringRef("seed scope"),
	})
	require.NoError(t, err)
	require.NotNil(t, sc)

	negative := constants.PolicyLogicNegative
	policy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name:        "deny-" + role + "-" + uuid.New().String(),
		Description: refs.NewStringRef("seed deny policy"),
		Type:        constants.PolicyTypeRole,
		Logic:       &negative,
		Targets: []*model.PolicyTargetInput{
			{
				TargetType:  constants.TargetTypeRole,
				TargetValue: role,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, policy)
	require.Equal(t, constants.PolicyLogicNegative, policy.Logic, "policy must be stored as negative")

	perm, err := ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       resource + "-" + scope,
		ResourceID: res.ID,
		ScopeIds:   []string{sc.ID},
		PolicyIds:  []string{policy.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, perm)
}

// TestCheckPermission_ResultLabels_IncrementCorrectCounter covers the four
// result labels on authorizer_authz_checks_total that the earlier rollout test
// (TestCheckPermission_IncrementsPrometheusCounters) did not exercise:
// allowed, denied, unmatched_denied, and error. Each subtest builds the exact
// shape needed to land on one terminal path in CheckPermission and asserts
// that exactly one increment is recorded on the matching counter series.
// Co-located with the authz tests because it shares their fixtures.
func TestCheckPermission_ResultLabels_IncrementCorrectCounter(t *testing.T) {
	t.Run("allowed", func(t *testing.T) {
		ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
		req, ctx := createContext(ts)
		adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

		// Seed a granting role policy + permission for (orders, read) and a
		// user principal with the "user" role so the affirmative grant fires.
		seedResourceScopePermissionAllowingRole(t, ts, ctx, "orders", "read", "user")

		before := testutil.ToFloat64(metrics.AuthzChecksTotal.WithLabelValues(
			metrics.AuthzModeEnforcing, metrics.AuthzResultAllowed))

		res, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
			ID: "user-allowed", Type: constants.PrincipalTypeUser, Roles: []string{"user"},
		}, "orders", "read")
		require.NoError(t, err)
		require.True(t, res.Allowed)

		after := testutil.ToFloat64(metrics.AuthzChecksTotal.WithLabelValues(
			metrics.AuthzModeEnforcing, metrics.AuthzResultAllowed))
		require.Equal(t, before+1, after, "allowed counter must increment once")
	})

	t.Run("denied", func(t *testing.T) {
		ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
		req, ctx := createContext(ts)
		adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

		// Negative-logic policy targeting "user" — any principal with that
		// role is explicitly denied on (orders, read).
		seedResourceScopePermissionWithDenyPolicy(t, ts, ctx, "orders", "read", "user")

		before := testutil.ToFloat64(metrics.AuthzChecksTotal.WithLabelValues(
			metrics.AuthzModeEnforcing, metrics.AuthzResultDenied))

		res, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
			ID: "user-denied", Type: constants.PrincipalTypeUser, Roles: []string{"user"},
		}, "orders", "read")
		require.NoError(t, err)
		require.False(t, res.Allowed)

		after := testutil.ToFloat64(metrics.AuthzChecksTotal.WithLabelValues(
			metrics.AuthzModeEnforcing, metrics.AuthzResultDenied))
		require.Equal(t, before+1, after, "denied counter must increment once")
	})

	t.Run("unmatched_denied", func(t *testing.T) {
		ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
		_, ctx := createContext(ts)

		// Known (resource, scope) with no permission row in enforcing mode →
		// the fall-through denies and increments unmatched_denied.
		seedKnownResourceScopeNoPermission(t, ts, ctx, "orders", "read")

		before := testutil.ToFloat64(metrics.AuthzChecksTotal.WithLabelValues(
			metrics.AuthzModeEnforcing, metrics.AuthzResultUnmatchedDenied))

		res, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
			ID: "user-unmatched", Type: constants.PrincipalTypeUser,
		}, "orders", "read")
		require.NoError(t, err)
		require.False(t, res.Allowed)

		after := testutil.ToFloat64(metrics.AuthzChecksTotal.WithLabelValues(
			metrics.AuthzModeEnforcing, metrics.AuthzResultUnmatchedDenied))
		require.Equal(t, before+1, after, "unmatched_denied counter must increment once")
	})

	t.Run("error", func(t *testing.T) {
		ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
		_, ctx := createContext(ts)

		before := testutil.ToFloat64(metrics.AuthzChecksTotal.WithLabelValues(
			metrics.AuthzModeEnforcing, metrics.AuthzResultError))

		// Invalid identifier — fails the input validation path which records
		// AuthzResultError before any storage or cache lookup.
		_, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
			ID: "user-error", Type: constants.PrincipalTypeUser,
		}, "bad resource with spaces", "read")
		require.Error(t, err)

		after := testutil.ToFloat64(metrics.AuthzChecksTotal.WithLabelValues(
			metrics.AuthzModeEnforcing, metrics.AuthzResultError))
		require.Equal(t, before+1, after, "error counter must increment once")
	})
}

// seedResourceScopePermissionAllowingRole seeds a resource + scope and an
// affirmative-logic role policy targeting `role`, then attaches a permission
// linking them. Mirrors seedResourceScopePermissionWithDenyPolicy but for the
// grant path.
func seedResourceScopePermissionAllowingRole(
	t *testing.T,
	ts *testSetup,
	ctx context.Context,
	resource, scope, role string,
) {
	t.Helper()

	res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{
		Name:        resource,
		Description: refs.NewStringRef("seed resource"),
	})
	require.NoError(t, err)

	sc, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{
		Name:        scope,
		Description: refs.NewStringRef("seed scope"),
	})
	require.NoError(t, err)

	pol, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name:        "allow-" + role + "-" + uuid.New().String()[:8],
		Description: refs.NewStringRef("seed allow policy"),
		Type:        constants.PolicyTypeRole,
		Logic:       refs.NewStringRef(constants.PolicyLogicPositive),
		Targets: []*model.PolicyTargetInput{
			{TargetType: constants.TargetTypeRole, TargetValue: role},
		},
	})
	require.NoError(t, err)

	perm, err := ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:        "allow-" + resource + "-" + scope + "-" + uuid.New().String()[:8],
		Description: refs.NewStringRef("seed allow permission"),
		ResourceID:  res.ID,
		ScopeIds:    []string{sc.ID},
		PolicyIds:   []string{pol.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, perm)
}

// TestConfig_LegacyDisabledMigration_NormalizesToPermissive verifies that the
// legacy "disabled" enforcement value is migrated to "permissive". The actual
// one-time migration log is emitted by runRoot (see cmd/root.go); here we only
// assert the canonical value produced by the normalizer.
func TestConfig_LegacyDisabledMigration_NormalizesToPermissive(t *testing.T) {
	cfg := &config.Config{AuthorizationEnforcement: "disabled"}
	migrated := cmd.NormalizeAuthzEnforcement(cfg.AuthorizationEnforcement)
	require.Equal(t, constants.AuthorizationEnforcementPermissive, migrated)
}

// TestConfig_EmptyValue_NormalizesToPermissive verifies the empty string (flag
// unset) maps to the new default.
func TestConfig_EmptyValue_NormalizesToPermissive(t *testing.T) {
	require.Equal(t, constants.AuthorizationEnforcementPermissive, cmd.NormalizeAuthzEnforcement(""))
}

// TestConfig_UnknownValue_NormalizesToPermissive verifies unrecognized input is
// mapped to the safe default ("permissive") rather than propagated.
func TestConfig_UnknownValue_NormalizesToPermissive(t *testing.T) {
	require.Equal(t, constants.AuthorizationEnforcementPermissive, cmd.NormalizeAuthzEnforcement("banana"))
}

// TestConfig_Enforcing_Preserved verifies "enforcing" passes through unchanged.
func TestConfig_Enforcing_Preserved(t *testing.T) {
	require.Equal(t, constants.AuthorizationEnforcementEnforcing, cmd.NormalizeAuthzEnforcement("enforcing"))
}

// TestConfig_MixedCaseEnforcing_Preserved verifies the normalizer is tolerant
// of mixed case, uppercase, and surrounding whitespace on "enforcing". This
// guards against a silent demotion to permissive when an operator types
// `--authorization-enforcement Enforcing` or sets `ENFORCING` via CI.
func TestConfig_MixedCaseEnforcing_Preserved(t *testing.T) {
	require.Equal(t, constants.AuthorizationEnforcementEnforcing, cmd.NormalizeAuthzEnforcement("Enforcing"))
	require.Equal(t, constants.AuthorizationEnforcementEnforcing, cmd.NormalizeAuthzEnforcement("ENFORCING"))
	require.Equal(t, constants.AuthorizationEnforcementEnforcing, cmd.NormalizeAuthzEnforcement("  enforcing  "))
}

// seedResourceScopePermissionWithRolePolicy seeds a resource, scope, a
// role policy with the given logic targeting the given role, and a permission
// that links them. Shared implementation used by the positive- and
// negative-logic helpers.
func seedResourceScopePermissionWithRolePolicy(
	t *testing.T,
	ts *testSetup,
	ctx context.Context,
	resource, scope, role, logic string,
) {
	t.Helper()

	res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{
		Name:        resource,
		Description: refs.NewStringRef("seed resource"),
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	sc, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{
		Name:        scope,
		Description: refs.NewStringRef("seed scope"),
	})
	require.NoError(t, err)
	require.NotNil(t, sc)

	logicRef := logic
	policy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name:        logic + "-" + role + "-" + uuid.New().String(),
		Description: refs.NewStringRef("seed role policy"),
		Type:        constants.PolicyTypeRole,
		Logic:       &logicRef,
		Targets: []*model.PolicyTargetInput{
			{
				TargetType:  constants.TargetTypeRole,
				TargetValue: role,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, policy)
	require.Equal(t, logic, policy.Logic, "policy must be stored with requested logic")

	perm, err := ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       resource + "-" + scope + "-" + uuid.New().String(),
		ResourceID: res.ID,
		ScopeIds:   []string{sc.ID},
		PolicyIds:  []string{policy.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, perm)
}

// seedResourceScopePermissionWithPositivePolicy seeds a resource, scope, a
// positive-logic role policy targeting the given role, and a permission that
// links them. Mirrors seedResourceScopePermissionWithDenyPolicy but with grant
// semantics.
func seedResourceScopePermissionWithPositivePolicy(
	t *testing.T,
	ts *testSetup,
	ctx context.Context,
	resource, scope, role string,
) {
	t.Helper()
	seedResourceScopePermissionWithRolePolicy(t, ts, ctx, resource, scope, role, constants.PolicyLogicPositive)
}

// seedResourceScopeWithUnanimousDualRolePolicy seeds a resource, scope, TWO
// positive-logic role policies (one per role), and a permission that links
// them with DecisionStrategy=unanimous. This is the minimal setup to exercise
// the unanimous evaluation path (all attached policies must agree).
func seedResourceScopeWithUnanimousDualRolePolicy(
	t *testing.T,
	ts *testSetup,
	ctx context.Context,
	resource, scope, roleA, roleB string,
) {
	t.Helper()

	res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{
		Name:        resource,
		Description: refs.NewStringRef("seed resource"),
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	sc, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{
		Name:        scope,
		Description: refs.NewStringRef("seed scope"),
	})
	require.NoError(t, err)
	require.NotNil(t, sc)

	positive := constants.PolicyLogicPositive

	policyA, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name:        "grant-" + roleA + "-" + uuid.New().String(),
		Description: refs.NewStringRef("seed positive role policy A"),
		Type:        constants.PolicyTypeRole,
		Logic:       &positive,
		Targets: []*model.PolicyTargetInput{
			{
				TargetType:  constants.TargetTypeRole,
				TargetValue: roleA,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, policyA)

	policyB, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name:        "grant-" + roleB + "-" + uuid.New().String(),
		Description: refs.NewStringRef("seed positive role policy B"),
		Type:        constants.PolicyTypeRole,
		Logic:       &positive,
		Targets: []*model.PolicyTargetInput{
			{
				TargetType:  constants.TargetTypeRole,
				TargetValue: roleB,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, policyB)

	unanimous := constants.DecisionStrategyUnanimous
	perm, err := ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:             resource + "-" + scope + "-" + uuid.New().String(),
		ResourceID:       res.ID,
		ScopeIds:         []string{sc.ID},
		PolicyIds:        []string{policyA.ID, policyB.ID},
		DecisionStrategy: &unanimous,
	})
	require.NoError(t, err)
	require.NotNil(t, perm)
	require.Equal(t, constants.DecisionStrategyUnanimous, perm.DecisionStrategy,
		"permission must be persisted with unanimous strategy")
}

// seedResourceScopeWithUserPolicyPermission seeds a resource, scope, a
// positive-logic user policy targeting the given userID, and a permission that
// links them. Exercises the PolicyTypeUser path: the policy matches on
// principal.ID, not roles.
func seedResourceScopeWithUserPolicyPermission(
	t *testing.T,
	ts *testSetup,
	ctx context.Context,
	resource, scope, userID string,
) {
	t.Helper()

	res, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{
		Name:        resource,
		Description: refs.NewStringRef("seed resource"),
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	sc, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{
		Name:        scope,
		Description: refs.NewStringRef("seed scope"),
	})
	require.NoError(t, err)
	require.NotNil(t, sc)

	positive := constants.PolicyLogicPositive
	policy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name:        "user-grant-" + uuid.New().String(),
		Description: refs.NewStringRef("seed user policy"),
		Type:        constants.PolicyTypeUser,
		Logic:       &positive,
		Targets: []*model.PolicyTargetInput{
			{
				TargetType:  constants.TargetTypeUser,
				TargetValue: userID,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, policy)

	perm, err := ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       resource + "-" + scope + "-" + uuid.New().String(),
		ResourceID: res.ID,
		ScopeIds:   []string{sc.ID},
		PolicyIds:  []string{policy.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, perm)
}

// TestCheckPermission_MaxScopes_InsideCeiling_UsesPolicy verifies that when a
// principal's delegation ceiling (MaxScopes) explicitly includes the requested
// resource:scope, the normal policy evaluation proceeds and a matching
// positive policy still grants access.
func TestCheckPermission_MaxScopes_InsideCeiling_UsesPolicy(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	req, ctx := createContext(ts)
	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	seedResourceScopePermissionWithPositivePolicy(t, ts, ctx, "docs", "read", "viewer")

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:        "user-1",
		Type:      constants.PrincipalTypeUser,
		Roles:     []string{"viewer"},
		MaxScopes: []string{"docs:read"},
	}, "docs", "read")
	require.NoError(t, err)
	require.True(t, result.Allowed)
}

// TestCheckPermission_MaxScopes_OutsideCeiling_DeniesBeforePolicy verifies that
// even when a principal's roles/policies would normally grant access, a
// MaxScopes ceiling that does not include the requested resource:scope MUST
// deny the check short-circuit — delegation ceilings are evaluated before
// policy matching.
func TestCheckPermission_MaxScopes_OutsideCeiling_DeniesBeforePolicy(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	req, ctx := createContext(ts)
	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	seedResourceScopePermissionWithPositivePolicy(t, ts, ctx, "docs", "read", "viewer")

	result, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:        "user-1",
		Type:      constants.PrincipalTypeUser,
		Roles:     []string{"viewer"},
		MaxScopes: []string{"docs:write"},
	}, "docs", "read")
	require.NoError(t, err)
	require.False(t, result.Allowed, "MaxScopes ceiling must deny before policy eval")
}

// TestCheckPermission_UnanimousDecisionStrategy_AllPoliciesMustAgree verifies
// that a permission with DecisionStrategy=unanimous only grants when every
// attached policy's target matches the principal. A principal with only one of
// the two required roles must be denied; a principal with both is allowed.
func TestCheckPermission_UnanimousDecisionStrategy_AllPoliciesMustAgree(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	req, ctx := createContext(ts)
	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	seedResourceScopeWithUnanimousDualRolePolicy(t, ts, ctx, "ledger", "read", "accountant", "auditor")

	res, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:    "user-1",
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"accountant"},
	}, "ledger", "read")
	require.NoError(t, err)
	require.False(t, res.Allowed, "unanimous: missing one role")

	res2, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:    "user-2",
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"accountant", "auditor"},
	}, "ledger", "read")
	require.NoError(t, err)
	require.True(t, res2.Allowed, "unanimous: all roles present")
}

// TestCheckPermission_UserTypePolicy_MatchesOnPrincipalID verifies that a
// PolicyTypeUser policy matches the principal by its ID (not by role). The
// seeded policy grants access to a specific user; any other user must be
// denied.
func TestCheckPermission_UserTypePolicy_MatchesOnPrincipalID(t *testing.T) {
	ts := testSetupWithAuthzMode(t, constants.AuthorizationEnforcementEnforcing)
	req, ctx := createContext(ts)
	adminHash, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	seedResourceScopeWithUserPolicyPermission(t, ts, ctx, "secret", "read", "user-alice")

	res, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:   "user-alice",
		Type: constants.PrincipalTypeUser,
	}, "secret", "read")
	require.NoError(t, err)
	require.True(t, res.Allowed)

	res2, err := ts.Authz.CheckPermission(ctx, &authorization.Principal{
		ID:   "user-bob",
		Type: constants.PrincipalTypeUser,
	}, "secret", "read")
	require.NoError(t, err)
	require.False(t, res2.Allowed)
}
