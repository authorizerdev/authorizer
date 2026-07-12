package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestUserOrganizations tests the _user_organizations admin query.
func TestUserOrganizations(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	userID := seedOrgTestUser(t, ts)

	t.Run("should reject non-super-admin caller", func(t *testing.T) {
		res, err := ts.GraphQLProvider.UserOrganizations(ctx, &model.UserOrganizationsRequest{UserID: userID})
		require.Error(t, err)
		require.Nil(t, res)
	})

	// Everything below runs as super-admin.
	setAdminCookie(t, ts)

	t.Run("should require a user_id", func(t *testing.T) {
		res, err := ts.GraphQLProvider.UserOrganizations(ctx, &model.UserOrganizationsRequest{UserID: "  "})
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("should return an empty list for a user with no organizations", func(t *testing.T) {
		res, err := ts.GraphQLProvider.UserOrganizations(ctx, &model.UserOrganizationsRequest{UserID: userID})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Len(t, res.UserOrganizations, 0)
		assert.NotNil(t, res.Pagination)
	})

	t.Run("should list the organizations a user belongs to with per-org roles", func(t *testing.T) {
		orgA, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: "user-orgs-a-" + uuid.NewString(),
		})
		require.NoError(t, err)
		orgB, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: "user-orgs-b-" + uuid.NewString(),
		})
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  orgA.ID,
			UserID: userID,
			Roles:  []string{"admin", "billing"},
		})
		require.NoError(t, err)
		_, err = ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  orgB.ID,
			UserID: userID,
			Roles:  []string{"member"},
		})
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.UserOrganizations(ctx, &model.UserOrganizationsRequest{UserID: userID})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.UserOrganizations, 2)

		rolesByOrg := map[string][]string{}
		for _, uo := range res.UserOrganizations {
			require.NotNil(t, uo.Organization)
			rolesByOrg[uo.Organization.ID] = uo.Roles
		}
		assert.ElementsMatch(t, []string{"admin", "billing"}, rolesByOrg[orgA.ID])
		assert.ElementsMatch(t, []string{"member"}, rolesByOrg[orgB.ID])
	})
}
