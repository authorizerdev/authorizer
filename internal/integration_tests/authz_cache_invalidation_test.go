package integration_tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestAuthzCacheInvalidation_OnAdminMutations verifies that the memory_store-
// backed decision cache is actually invalidated when the policy graph changes
// via admin mutations. The test enables the authz cache (CacheTTL > 0),
// primes a deny verdict for (resource, scope), then mutates the graph via
// each admin operation that should invalidate cache, and confirms the next
// check produces the new verdict instead of the cached one.
func TestAuthzCacheInvalidation_OnAdminMutations(t *testing.T) {
	cfg := getTestConfig()
	cfg.AuthorizationCacheTTL = 300 // turn the cache ON for this test
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	adminHash, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))
	t.Cleanup(func() { req.Header.Del("Cookie") })

	resource, err := ts.GraphQLProvider.AuthzAddResource(ctx, &model.AddResourceInput{Name: "inv-docs"})
	require.NoError(t, err)
	readScope, err := ts.GraphQLProvider.AuthzAddScope(ctx, &model.AddScopeInput{Name: "inv-read"})
	require.NoError(t, err)
	writeScope, err := ts.GraphQLProvider.AuthzAddScope(ctx, &model.AddScopeInput{Name: "inv-write"})
	require.NoError(t, err)
	policy, err := ts.GraphQLProvider.AuthzAddPolicy(ctx, &model.AddPolicyInput{
		Name: "inv-user-role",
		Type: "role",
		Targets: []*model.PolicyTargetInput{
			{TargetType: "role", TargetValue: "user"},
		},
	})
	require.NoError(t, err)

	principal := &authorization.Principal{
		ID:    uuid.New().String(),
		Type:  constants.PrincipalTypeUser,
		Roles: []string{"user"},
	}

	check := func(t *testing.T, scope string) bool {
		t.Helper()
		res, err := ts.Authz.CheckPermission(context.Background(), principal, "inv-docs", scope)
		require.NoError(t, err)
		require.NotNil(t, res)
		return res.Allowed
	}

	t.Run("add permission flips cached deny to allow", func(t *testing.T) {
		// Prime: no permission row exists yet, evaluator returns deny and
		// memory_store now holds "false" for this (principal, resource, scope).
		assert.False(t, check(t, "inv-read"), "must deny before any permission exists")

		// Mutate: add a permission granting inv-docs:inv-read via the
		// user-role policy. The graphql layer calls InvalidateCache.
		perm, err := ts.GraphQLProvider.AuthzAddPermission(ctx, &model.AddPermissionInput{
			Name:       "inv-docs-read",
			ResourceID: resource.ID,
			ScopeIds:   []string{readScope.ID},
			PolicyIds:  []string{policy.ID},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = ts.GraphQLProvider.AuthzDeletePermission(ctx, perm.ID)
		})

		// Verdict must update — if the cache wasn't invalidated, we'd still
		// see the stale "false".
		assert.True(t, check(t, "inv-read"), "cached deny must be invalidated; new permission should grant access")
	})

	t.Run("update permission swap-scopes flips verdicts", func(t *testing.T) {
		// Seed a permission for inv-read, prime cache (allow for inv-read,
		// deny for inv-write).
		perm, err := ts.GraphQLProvider.AuthzAddPermission(ctx, &model.AddPermissionInput{
			Name:       "inv-update-perm",
			ResourceID: resource.ID,
			ScopeIds:   []string{readScope.ID},
			PolicyIds:  []string{policy.ID},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = ts.GraphQLProvider.AuthzDeletePermission(ctx, perm.ID)
		})

		require.True(t, check(t, "inv-read"), "precondition: inv-read allowed via new permission")
		require.False(t, check(t, "inv-write"), "precondition: inv-write denied (not in any permission row)")

		// Swap the permission's scope set from read → write. Cache for both
		// pairs must be invalidated.
		_, err = ts.GraphQLProvider.AuthzUpdatePermission(ctx, &model.UpdatePermissionInput{
			ID:       perm.ID,
			ScopeIds: []string{writeScope.ID},
		})
		require.NoError(t, err)

		assert.False(t, check(t, "inv-read"), "cached allow must be invalidated after scope removed")
		assert.True(t, check(t, "inv-write"), "cached deny must be invalidated after scope added")
	})

	t.Run("delete permission flips cached allow back to deny", func(t *testing.T) {
		perm, err := ts.GraphQLProvider.AuthzAddPermission(ctx, &model.AddPermissionInput{
			Name:       "inv-delete-perm",
			ResourceID: resource.ID,
			ScopeIds:   []string{readScope.ID},
			PolicyIds:  []string{policy.ID},
		})
		require.NoError(t, err)
		require.True(t, check(t, "inv-read"), "precondition: inv-read allowed before delete")

		_, err = ts.GraphQLProvider.AuthzDeletePermission(ctx, perm.ID)
		require.NoError(t, err)

		assert.False(t, check(t, "inv-read"), "cached allow must be invalidated after permission deletion")
	})

	t.Run("delete resource invalidates downstream cache", func(t *testing.T) {
		// Fresh resource so the deletion doesn't disturb the outer fixtures.
		tmpResource, err := ts.GraphQLProvider.AuthzAddResource(ctx, &model.AddResourceInput{Name: "inv-tmp"})
		require.NoError(t, err)
		perm, err := ts.GraphQLProvider.AuthzAddPermission(ctx, &model.AddPermissionInput{
			Name:       "inv-tmp-perm",
			ResourceID: tmpResource.ID,
			ScopeIds:   []string{readScope.ID},
			PolicyIds:  []string{policy.ID},
		})
		require.NoError(t, err)

		res, err := ts.Authz.CheckPermission(context.Background(), principal, "inv-tmp", "inv-read")
		require.NoError(t, err)
		require.True(t, res.Allowed, "precondition: granted before resource delete")

		// Cascade-delete the permission first (Postgres FK + Mongo lookup
		// reasons), then drop the resource.
		_, err = ts.GraphQLProvider.AuthzDeletePermission(ctx, perm.ID)
		require.NoError(t, err)
		_, err = ts.GraphQLProvider.AuthzDeleteResource(ctx, tmpResource.ID)
		require.NoError(t, err)

		// Re-check — the cached allow must be gone (DeletePermission already
		// invalidates; this assertion guards against a future regression
		// where DeleteResource forgets to invalidate.)
		res, err = ts.Authz.CheckPermission(context.Background(), principal, "inv-tmp", "inv-read")
		require.NoError(t, err)
		assert.False(t, res.Allowed, "cached allow must be invalidated after resource deletion")
	})
}
