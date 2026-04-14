package integration_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
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

	// CheckPermission requires an authenticated user, not admin.
	// Sign up a user, log in, and set the access token.
	t.Run("should check permission granted by role", func(t *testing.T) {
		// Clear admin cookie
		req.Header.Del("Cookie")

		email := "authz_test_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"

		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)

		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &email,
			Password: password,
		})
		require.NoError(t, err)
		require.NotNil(t, loginRes)
		require.NotNil(t, loginRes.User)

		// User has "user" role by default (from DefaultRoles in config).
		// Set access token for the CheckPermission call.
		allData, err := ts.MemoryStoreProvider.GetAllData()
		require.NoError(t, err)
		accessToken := ""
		for k, v := range allData {
			if strings.Contains(k, constants.TokenTypeAccessToken) {
				accessToken = v
				break
			}
		}
		require.NotEmpty(t, accessToken, "access token must be present in memory store")

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		defer func() {
			req.Header.Del("Authorization")
			// Restore admin cookie for subsequent tests
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))
		}()

		res, err := ts.GraphQLProvider.CheckPermission(ctx, &model.CheckPermissionInput{
			Resource: "documents",
			Scope:    "read",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Allowed, "user with 'user' role should have read access to documents")
	})

	t.Run("should check permission denied for wrong role", func(t *testing.T) {
		// Create a second policy that targets "admin" role only
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

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

		// Add a "write" scope
		writeScope, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{
			Name: "write",
		})
		require.NoError(t, err)

		// Create a permission that requires "admin" role for "write" on documents
		_, err = ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
			Name:       "documents-write",
			ResourceID: resourceID,
			ScopeIds:   []string{writeScope.ID},
			PolicyIds:  []string{adminPolicy.ID},
		})
		require.NoError(t, err)

		// Now sign up a regular user and check write permission (should be denied)
		req.Header.Del("Cookie")

		email2 := "authz_denied_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"

		_, err = ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email2,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)

		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &email2,
			Password: password,
		})
		require.NoError(t, err)
		require.NotNil(t, loginRes)

		allData, err := ts.MemoryStoreProvider.GetAllData()
		require.NoError(t, err)
		accessToken := ""
		for k, v := range allData {
			if strings.Contains(k, constants.TokenTypeAccessToken) {
				accessToken = v
				break
			}
		}
		require.NotEmpty(t, accessToken)

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		defer func() {
			req.Header.Del("Authorization")
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))
		}()

		res, err := ts.GraphQLProvider.CheckPermission(ctx, &model.CheckPermissionInput{
			Resource: "documents",
			Scope:    "write",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Allowed, "user with 'user' role should NOT have write access requiring admin role")
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
