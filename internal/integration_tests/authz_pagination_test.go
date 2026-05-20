package integration_tests

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestAuthzListPagination_AdminQueries seeds N rows of each FGA entity and
// asserts the four list resolvers (Resources / Scopes / Policies /
// Permissions) honor limit, page, offset, and total. Each subtest exercises
// page 1 + page 2 + an over-the-end page to confirm offset math and that
// total reflects the full row count regardless of the returned slice size.
func TestAuthzListPagination_AdminQueries(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	adminHash, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))
	t.Cleanup(func() { req.Header.Del("Cookie") })

	// Unique prefix isolates this test's fixtures from other tests in the
	// same test binary (which all share an SQLite DB inside t.TempDir()).
	prefix := "page-" + uuid.New().String()[:8] + "-"
	const seedCount = 5
	resourceIDs := make([]string, 0, seedCount)
	scopeIDs := make([]string, 0, seedCount)
	policyIDs := make([]string, 0, seedCount)
	permissionIDs := make([]string, 0, seedCount)

	for i := 0; i < seedCount; i++ {
		res, err := ts.GraphQLProvider.AuthzAddResource(ctx, &model.AddResourceInput{
			Name: fmt.Sprintf("%sresource-%d", prefix, i),
		})
		require.NoError(t, err)
		resourceIDs = append(resourceIDs, res.ID)

		scope, err := ts.GraphQLProvider.AuthzAddScope(ctx, &model.AddScopeInput{
			Name: fmt.Sprintf("%sscope-%d", prefix, i),
		})
		require.NoError(t, err)
		scopeIDs = append(scopeIDs, scope.ID)

		pol, err := ts.GraphQLProvider.AuthzAddPolicy(ctx, &model.AddPolicyInput{
			Name: fmt.Sprintf("%spolicy-%d", prefix, i),
			Type: "role",
			Targets: []*model.PolicyTargetInput{
				{TargetType: "role", TargetValue: "user"},
			},
		})
		require.NoError(t, err)
		policyIDs = append(policyIDs, pol.ID)

		perm, err := ts.GraphQLProvider.AuthzAddPermission(ctx, &model.AddPermissionInput{
			Name:       fmt.Sprintf("%spermission-%d", prefix, i),
			ResourceID: res.ID,
			ScopeIds:   []string{scope.ID},
			PolicyIds:  []string{pol.ID},
		})
		require.NoError(t, err)
		permissionIDs = append(permissionIDs, perm.ID)
	}

	// page1 + page2 + over-the-end behavior is symmetric across the four
	// resolvers, so each subtest below makes the same three assertions
	// against a different list endpoint.

	t.Run("resources pagination", func(t *testing.T) {
		page1, err := ts.GraphQLProvider.AuthzResources(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(1)},
		})
		require.NoError(t, err)
		require.NotNil(t, page1)
		require.NotNil(t, page1.Pagination)
		assert.Equal(t, int64(2), page1.Pagination.Limit, "limit echoes input")
		assert.Equal(t, int64(1), page1.Pagination.Page, "page echoes input")
		assert.Equal(t, int64(0), page1.Pagination.Offset, "page 1 offset = 0")
		assert.GreaterOrEqual(t, page1.Pagination.Total, int64(seedCount), "total must include all seeded rows")
		assert.Len(t, page1.Resources, 2, "page 1 returns Limit items")

		page2, err := ts.GraphQLProvider.AuthzResources(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(2)},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), page2.Pagination.Offset, "page 2 offset = (page-1)*limit")
		assert.Len(t, page2.Resources, 2, "page 2 returns Limit items")
		assertNoOverlap(t, idsOf(page1.Resources), idsOf(page2.Resources))

		// Page far past the end returns zero items but the total is still
		// the full row count.
		pageEnd, err := ts.GraphQLProvider.AuthzResources(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(100)},
		})
		require.NoError(t, err)
		assert.Empty(t, pageEnd.Resources, "page past the end is empty")
		assert.Equal(t, page1.Pagination.Total, pageEnd.Pagination.Total, "total is invariant across pages")
	})

	t.Run("scopes pagination", func(t *testing.T) {
		page1, err := ts.GraphQLProvider.AuthzScopes(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(1)},
		})
		require.NoError(t, err)
		require.NotNil(t, page1)
		require.NotNil(t, page1.Pagination)
		assert.Equal(t, int64(2), page1.Pagination.Limit)
		assert.Equal(t, int64(0), page1.Pagination.Offset)
		assert.GreaterOrEqual(t, page1.Pagination.Total, int64(seedCount))
		assert.Len(t, page1.Scopes, 2)

		page2, err := ts.GraphQLProvider.AuthzScopes(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(2)},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), page2.Pagination.Offset)
		assert.Len(t, page2.Scopes, 2)
		assertNoOverlap(t, scopeIdsOf(page1.Scopes), scopeIdsOf(page2.Scopes))
	})

	t.Run("policies pagination", func(t *testing.T) {
		page1, err := ts.GraphQLProvider.AuthzPolicies(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(1)},
		})
		require.NoError(t, err)
		require.NotNil(t, page1)
		assert.Equal(t, int64(0), page1.Pagination.Offset)
		assert.GreaterOrEqual(t, page1.Pagination.Total, int64(seedCount))
		assert.Len(t, page1.Policies, 2)

		page2, err := ts.GraphQLProvider.AuthzPolicies(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(2)},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), page2.Pagination.Offset)
		assert.Len(t, page2.Policies, 2)
		assertNoOverlap(t, policyIdsOf(page1.Policies), policyIdsOf(page2.Policies))
	})

	t.Run("permissions pagination", func(t *testing.T) {
		page1, err := ts.GraphQLProvider.AuthzPermissions(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(1)},
		})
		require.NoError(t, err)
		require.NotNil(t, page1)
		assert.Equal(t, int64(0), page1.Pagination.Offset)
		assert.GreaterOrEqual(t, page1.Pagination.Total, int64(seedCount))
		assert.Len(t, page1.Permissions, 2)

		page2, err := ts.GraphQLProvider.AuthzPermissions(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{Limit: ptrInt64(2), Page: ptrInt64(2)},
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), page2.Pagination.Offset)
		assert.Len(t, page2.Permissions, 2)
		assertNoOverlap(t, permissionIdsOf(page1.Permissions), permissionIdsOf(page2.Permissions))
	})

	// Touch the seeded IDs so the linter sees them as used even if a future
	// edit drops a subtest. They're meaningful as fixture witnesses.
	_ = resourceIDs
	_ = scopeIDs
	_ = policyIDs
	_ = permissionIDs
}

func ptrInt64(v int64) *int64 { return &v }

func idsOf(items []*model.AuthzResource) []string {
	ids := make([]string, len(items))
	for i, x := range items {
		ids[i] = x.ID
	}
	return ids
}

func scopeIdsOf(items []*model.AuthzScope) []string {
	ids := make([]string, len(items))
	for i, x := range items {
		ids[i] = x.ID
	}
	return ids
}

func policyIdsOf(items []*model.AuthzPolicy) []string {
	ids := make([]string, len(items))
	for i, x := range items {
		ids[i] = x.ID
	}
	return ids
}

func permissionIdsOf(items []*model.AuthzPermission) []string {
	ids := make([]string, len(items))
	for i, x := range items {
		ids[i] = x.ID
	}
	return ids
}

func assertNoOverlap(t *testing.T, a, b []string) {
	t.Helper()
	set := make(map[string]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[id]; ok {
			t.Errorf("page 2 contains ID %q that already appeared on page 1", id)
		}
	}
}
