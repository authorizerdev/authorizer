package integration_tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// seedOrgTestUser inserts a basic-auth user directly via storage and returns
// its id. Org membership operations validate that the user exists.
func seedOrgTestUser(t *testing.T, ts *testSetup) string {
	t.Helper()
	id := uuid.New().String()
	email := "org-member-" + id + "@authorizer.test"
	now := int64(1)
	_, err := ts.StorageProvider.AddUser(context.Background(), &schemas.User{
		ID:              id,
		Email:           refs.NewStringRef(email),
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		Roles:           "user",
		EmailVerifiedAt: &now,
	})
	require.NoError(t, err)
	return id
}

func TestOrganizationAdmin(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("should reject non-super-admin caller", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: "unauthorized-" + uuid.NewString(),
		})
		require.Error(t, err)
		require.Nil(t, res)
	})

	// Everything below runs as super-admin.
	setAdminCookie(t, ts)

	t.Run("should reject empty/whitespace-only name", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: "   ",
		})
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("should create, get and list an organization", func(t *testing.T) {
		name := "acme-" + uuid.NewString()
		res, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name:        name,
			DisplayName: refs.NewStringRef("Acme Corporation"),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, name, res.Name)
		assert.True(t, res.Enabled)
		require.NotNil(t, res.DisplayName)
		assert.Equal(t, "Acme Corporation", *res.DisplayName)

		fetched, err := ts.GraphQLProvider.Organization(ctx, &model.OrganizationRequest{ID: res.ID})
		require.NoError(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, name, fetched.Name)

		list, err := ts.GraphQLProvider.Organizations(ctx, &model.ListOrganizationsRequest{})
		require.NoError(t, err)
		require.NotNil(t, list)
		assert.NotNil(t, list.Pagination)
		assert.Greater(t, len(list.Organizations), 0)
	})

	t.Run("should reject a duplicate organization name", func(t *testing.T) {
		name := "dup-" + uuid.NewString()
		_, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{Name: name})
		require.NoError(t, err)
		res, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{Name: name})
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("should add, list and remove an organization member", func(t *testing.T) {
		org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: "members-" + uuid.NewString(),
		})
		require.NoError(t, err)
		userID := seedOrgTestUser(t, ts)

		member, err := ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  org.ID,
			UserID: userID,
			Roles:  []string{"admin", "billing"},
		})
		require.NoError(t, err)
		require.NotNil(t, member)
		assert.Equal(t, org.ID, member.OrgID)
		assert.Equal(t, userID, member.UserID)
		assert.ElementsMatch(t, []string{"admin", "billing"}, member.Roles)

		// Duplicate membership rejected.
		dup, err := ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  org.ID,
			UserID: userID,
			Roles:  []string{"admin"},
		})
		require.Error(t, err)
		require.Nil(t, dup)

		members, err := ts.GraphQLProvider.OrgMembers(ctx, &model.ListOrgMembersRequest{OrgID: org.ID})
		require.NoError(t, err)
		require.NotNil(t, members)
		require.Len(t, members.OrgMembers, 1)
		assert.Equal(t, userID, members.OrgMembers[0].UserID)
		// The member's user identity is resolved for display.
		require.NotNil(t, members.OrgMembers[0].Email)
		assert.Equal(t, "org-member-"+userID+"@authorizer.test", *members.OrgMembers[0].Email)

		removed, err := ts.GraphQLProvider.RemoveOrgMember(ctx, &model.RemoveOrgMemberRequest{
			OrgID:  org.ID,
			UserID: userID,
		})
		require.NoError(t, err)
		require.NotNil(t, removed)

		after, err := ts.GraphQLProvider.OrgMembers(ctx, &model.ListOrgMembersRequest{OrgID: org.ID})
		require.NoError(t, err)
		assert.Len(t, after.OrgMembers, 0)
	})

	t.Run("should reject adding a member to a nonexistent org or a nonexistent user", func(t *testing.T) {
		org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: "validate-" + uuid.NewString(),
		})
		require.NoError(t, err)

		res, err := ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  uuid.NewString(),
			UserID: seedOrgTestUser(t, ts),
		})
		require.Error(t, err)
		require.Nil(t, res)

		res, err = ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  org.ID,
			UserID: uuid.NewString(),
		})
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("delete removes the organization and cascades its memberships", func(t *testing.T) {
		org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: "delete-me-" + uuid.NewString(),
		})
		require.NoError(t, err)
		userID := seedOrgTestUser(t, ts)
		_, err = ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  org.ID,
			UserID: userID,
		})
		require.NoError(t, err)

		delRes, err := ts.GraphQLProvider.DeleteOrganization(ctx, &model.OrganizationRequest{ID: org.ID})
		require.NoError(t, err)
		require.NotNil(t, delRes)

		_, err = ts.StorageProvider.GetOrganizationByID(ctx, org.ID)
		require.Error(t, err)

		_, err = ts.StorageProvider.GetOrgMembership(ctx, org.ID, userID)
		require.Error(t, err, "membership must be cascade-deleted with its organization")
	})
}
