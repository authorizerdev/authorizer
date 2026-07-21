package http_handlers

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	fgaengine "github.com/authorizerdev/authorizer/internal/authorization/engine/openfga"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// groupTestModel is the minimal ReBAC model the SCIM-group SAML projection needs:
// a group's members and a role's assignees may both be users or nested-group
// usersets (group#member).
const groupTestModel = `model
  schema 1.1
type user
type group
  relations
    define member: [user, group#member]
type role
  relations
    define assignee: [user, group#member]
`

// groupOnlyStore is a storage.Provider that answers only GetScimGroupByID (the
// one method assertedGroupsForOrg calls). Embedding the interface satisfies the
// full contract; any other call panics, which is the point — the test must not
// reach for anything else.
type groupOnlyStore struct {
	storage.Provider
	groups map[string]*schemas.ScimGroup
}

func (s groupOnlyStore) GetScimGroupByID(_ context.Context, id string) (*schemas.ScimGroup, error) {
	if g, ok := s.groups[id]; ok {
		return g, nil
	}
	return nil, assert.AnError
}

// TestAssertedGroupsCrossTenantContainment is the security-critical proof for
// this feature: a user who is (legitimately) a member of a group in Org B must
// NEVER have that group's name appear in an assertion issued for Org A.
func TestAssertedGroupsCrossTenantContainment(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	ctx := context.Background()

	eng, err := fgaengine.New(
		&fgaengine.Config{Store: fgaengine.StoreMemory, StoreName: "authorizer-groups-test"},
		&fgaengine.Dependencies{Log: &logger},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c, ok := eng.(interface{ Close() }); ok {
			c.Close()
		}
	})
	_, err = eng.WriteModel(ctx, groupTestModel)
	require.NoError(t, err)

	const (
		orgA   = "org-a"
		orgB   = "org-b"
		userID = "user-shared" // a real member of BOTH orgs
	)
	// Org A: "viewers". Org B: "admins" (the dangerous one).
	groupA := &schemas.ScimGroup{ID: "gid-a", OrgID: orgA, DisplayName: "viewers"}
	groupB := &schemas.ScimGroup{ID: "gid-b", OrgID: orgB, DisplayName: "admins"}
	store := groupOnlyStore{groups: map[string]*schemas.ScimGroup{
		groupA.ID: groupA,
		groupB.ID: groupB,
	}}

	// The user is a genuine member of a group in EACH org.
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + userID, Relation: "member", Object: "group:" + orgA + "/" + groupA.ID},
		{User: "user:" + userID, Relation: "member", Object: "group:" + orgB + "/" + groupB.ID},
	}))

	h := &httpProvider{Dependencies: Dependencies{
		Log:             &logger,
		StorageProvider: store,
		AuthzEngine:     eng,
	}}
	user := &schemas.User{ID: userID}

	// Assertion for Org A: only Org A's group name, NEVER Org B's "admins".
	groupsForA := h.assertedGroupsForOrg(ctx, orgA, user, &logger)
	assert.Equal(t, []string{"viewers"}, groupsForA)
	assert.NotContains(t, groupsForA, "admins", "cross-tenant leak: org-B group surfaced in an org-A assertion")

	// Assertion for Org B: only Org B's group name, NEVER Org A's "viewers".
	groupsForB := h.assertedGroupsForOrg(ctx, orgB, user, &logger)
	assert.Equal(t, []string{"admins"}, groupsForB)
	assert.NotContains(t, groupsForB, "viewers")

	// A user with no memberships in an org gets an empty set (fail-closed).
	groupsForStranger := h.assertedGroupsForOrg(ctx, orgA, &schemas.User{ID: "nobody"}, &logger)
	assert.Empty(t, groupsForStranger)
}

// TestAssertedGroupsGate2RejectsForgedNamespace proves the second, authoritative
// gate: even if a tuple's object id somehow carried this org's prefix but the
// stored row belongs to another org, the row-of-record check rejects it.
func TestAssertedGroupsGate2RejectsForgedNamespace(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	ctx := context.Background()

	eng, err := fgaengine.New(
		&fgaengine.Config{Store: fgaengine.StoreMemory, StoreName: "authorizer-groups-gate2"},
		&fgaengine.Dependencies{Log: &logger},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c, ok := eng.(interface{ Close() }); ok {
			c.Close()
		}
	})
	require.NoError(t, func() error { _, e := eng.WriteModel(ctx, groupTestModel); return e }())

	const orgA = "org-a"
	// The stored group row for "gid-x" actually belongs to org-b, but a tuple
	// places it under the org-a prefix (a forged/mismatched id).
	store := groupOnlyStore{groups: map[string]*schemas.ScimGroup{
		"gid-x": {ID: "gid-x", OrgID: "org-b", DisplayName: "admins"},
	}}
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:u", Relation: "member", Object: "group:" + orgA + "/gid-x"},
	}))

	h := &httpProvider{Dependencies: Dependencies{Log: &logger, StorageProvider: store, AuthzEngine: eng}}
	// Gate 1 (prefix) passes, but Gate 2 (row.OrgID == orgA) rejects it → no groups.
	got := h.assertedGroupsForOrg(ctx, orgA, &schemas.User{ID: "u"}, &logger)
	assert.Empty(t, got, "gate 2 must reject a group whose stored OrgID != issuing org")
}
