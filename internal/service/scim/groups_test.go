package scim

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fgaengine "github.com/authorizerdev/authorizer/internal/authorization/engine/openfga"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const groupSvcModel = `model
  schema 1.1
type user
type group
  relations
    define member: [user, group#member]
type role
  relations
    define assignee: [user, group#member]
`

// --- fakeStore ScimGroup methods (state in the groups map added to the struct). ---

func (f *fakeStore) AddScimGroup(_ context.Context, g *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	f.groups[g.ID] = g
	return g, nil
}

func (f *fakeStore) GetScimGroupByID(_ context.Context, id string) (*schemas.ScimGroup, error) {
	if g, ok := f.groups[id]; ok {
		return g, nil
	}
	return nil, errNotFound
}

func (f *fakeStore) GetScimGroupByOrgAndDisplayName(_ context.Context, orgID, displayName string) (*schemas.ScimGroup, error) {
	// Mirror the real storage contract: displayName is caseExact:false
	// (RFC 7644 §3.4.2.2), so the value is matched case-insensitively.
	for _, g := range f.groups {
		if g.OrgID == orgID && strings.EqualFold(g.DisplayName, displayName) {
			return g, nil
		}
	}
	return nil, errNotFound
}

func (f *fakeStore) GetScimGroupByOrgAndExternalID(_ context.Context, orgID, externalID string) (*schemas.ScimGroup, error) {
	want := orgID + ":" + externalID
	for _, g := range f.groups {
		if g.OrgID == orgID && g.ExternalID != nil && *g.ExternalID == want {
			return g, nil
		}
	}
	return nil, errNotFound
}

func (f *fakeStore) UpdateScimGroup(_ context.Context, g *schemas.ScimGroup) (*schemas.ScimGroup, error) {
	f.groups[g.ID] = g
	return g, nil
}

func (f *fakeStore) DeleteScimGroup(_ context.Context, g *schemas.ScimGroup) error {
	delete(f.groups, g.ID)
	return nil
}

// newGroupSvc builds a SCIM provider backed by a real embedded FGA engine.
func newGroupSvc(t *testing.T) (*provider, *fakeStore) {
	t.Helper()
	log := zerolog.New(zerolog.NewTestWriter(t))
	store := newFakeStore()
	eng, err := fgaengine.New(
		&fgaengine.Config{Store: fgaengine.StoreMemory, StoreName: "scim-groups-" + uuid.New().String()},
		&fgaengine.Dependencies{Log: &log},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c, ok := eng.(interface{ Close() }); ok {
			c.Close()
		}
	})
	_, err = eng.WriteModel(context.Background(), groupSvcModel)
	require.NoError(t, err)
	p := &provider{Dependencies: Dependencies{Log: &log, StorageProvider: store, AuthzEngine: eng}}
	return p, store
}

func TestGroupLifecycleAndMembership(t *testing.T) {
	p, store := newGroupSvc(t)
	ctx := context.Background()
	const org = "org-a"
	// Two org-a members + one member of a DIFFERENT org.
	store.memberships[org+"|u1"] = true
	store.memberships[org+"|u2"] = true
	store.memberships["org-b|intruder"] = true

	// Create.
	g, existed, err := p.CreateGroup(ctx, org, Group{DisplayName: "Engineers", ExternalID: "ext-1"})
	require.NoError(t, err)
	assert.False(t, existed)
	assert.Equal(t, "Engineers", g.DisplayName)

	// Idempotent create by externalId (same correlation key → same group).
	g2, existed2, err := p.CreateGroup(ctx, org, Group{DisplayName: "Engineers", ExternalID: "ext-1"})
	require.NoError(t, err)
	assert.True(t, existed2)
	assert.Equal(t, g.ID, g2.ID)

	// A create clashing on displayName with no matching externalId is a
	// uniqueness conflict (RFC 7644 §3.3 → 409), not a silent idempotent 200.
	_, _, err = p.CreateGroup(ctx, org, Group{DisplayName: "Engineers"})
	assert.ErrorIs(t, err, ErrGroupConflict)

	// Add u1, u2 (org members) AND intruder (org-b member — must be rejected).
	_, err = p.PatchGroup(ctx, org, g.ID, nil, nil, []MemberOp{
		{Op: "add", Members: []string{"u1", "u2", "intruder"}},
	})
	require.NoError(t, err)

	members, err := p.GroupMembers(ctx, org, g.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"u1", "u2"}, members, "cross-org member must not be added")
	assert.NotContains(t, members, "intruder")

	// Idempotent add (u1 again) — no duplicate, no error.
	_, err = p.PatchGroup(ctx, org, g.ID, nil, nil, []MemberOp{{Op: "add", Members: []string{"u1"}}})
	require.NoError(t, err)
	members, _ = p.GroupMembers(ctx, org, g.ID)
	assert.ElementsMatch(t, []string{"u1", "u2"}, members)

	// Remove u1 (Entra value-shape already normalised to MemberOp by the parser).
	_, err = p.PatchGroup(ctx, org, g.ID, nil, nil, []MemberOp{{Op: "remove", Members: []string{"u1"}}})
	require.NoError(t, err)
	members, _ = p.GroupMembers(ctx, org, g.ID)
	assert.ElementsMatch(t, []string{"u2"}, members)

	// Replace whole set → exactly {u1}.
	_, err = p.PatchGroup(ctx, org, g.ID, nil, nil, []MemberOp{{Op: "replace", Members: []string{"u1"}}})
	require.NoError(t, err)
	members, _ = p.GroupMembers(ctx, org, g.ID)
	assert.ElementsMatch(t, []string{"u1"}, members)

	// Clear ALL members via an unfiltered remove (deprovisioning) — must empty
	// the group, not no-op.
	_, err = p.PatchGroup(ctx, org, g.ID, nil, nil, []MemberOp{{Op: "remove", ClearAll: true}})
	require.NoError(t, err)
	members, _ = p.GroupMembers(ctx, org, g.ID)
	assert.Empty(t, members, "unfiltered remove must clear every member")

	// Re-add u1, u2 then clear via replace with an empty set → also empties.
	_, err = p.PatchGroup(ctx, org, g.ID, nil, nil, []MemberOp{{Op: "add", Members: []string{"u1", "u2"}}})
	require.NoError(t, err)
	_, err = p.PatchGroup(ctx, org, g.ID, nil, nil, []MemberOp{{Op: "replace", Members: nil}})
	require.NoError(t, err)
	members, _ = p.GroupMembers(ctx, org, g.ID)
	assert.Empty(t, members, "replace with an empty set must clear every member")

	// Rename via PATCH displayName.
	newName := "Platform"
	_, err = p.PatchGroup(ctx, org, g.ID, &newName, nil, nil)
	require.NoError(t, err)
	got, err := p.GetGroup(ctx, org, g.ID)
	require.NoError(t, err)
	assert.Equal(t, "Platform", got.DisplayName)

	// Delete removes the row and its membership tuples.
	require.NoError(t, p.DeleteGroup(ctx, org, g.ID))
	_, err = p.GetGroup(ctx, org, g.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestGroupDisplayNameCaseInsensitive proves that displayName (caseExact:false,
// RFC 7644 §3.4.2.2) is compared case-insensitively across the three paths that
// share GetScimGroupByOrgAndDisplayName: create-dedup, the rename-uniqueness
// gate, and the FindGroupByDisplayName probe.
func TestGroupDisplayNameCaseInsensitive(t *testing.T) {
	p, store := newGroupSvc(t)
	ctx := context.Background()
	const org = "org-a"
	store.memberships[org+"|u1"] = true
	store.memberships[org+"|u2"] = true

	// Create "Engineers" with an externalId + one member.
	g, existed, err := p.CreateGroup(ctx, org, Group{DisplayName: "Engineers", ExternalID: "ext-eng", Members: []string{"u1"}})
	require.NoError(t, err)
	assert.False(t, existed)

	// A create with a case-variant displayName and NO matching externalId is the
	// SAME group for uniqueness purposes → a 409 conflict, not a second row. This
	// is the intended reading of caseExact:false: "Engineers" and "engineers" are
	// one group.
	_, _, err = p.CreateGroup(ctx, org, Group{DisplayName: "engineers"})
	assert.ErrorIs(t, err, ErrGroupConflict, "case-variant create must conflict, not duplicate")
	assert.Len(t, store.groups, 1, "no duplicate row may be created for a case-variant name")

	// Same externalId + a case-variant displayName → idempotent adoption of the
	// existing row (existed=true, same id), member sync applied to THAT row, and
	// no duplicate created. The case-only rename passes the uniqueness gate
	// because it resolves to the group itself (self-exempt).
	adopted, existed, err := p.CreateGroup(ctx, org, Group{DisplayName: "engineers", ExternalID: "ext-eng", Members: []string{"u1", "u2"}})
	require.NoError(t, err)
	assert.True(t, existed, "case-variant name with the same externalId must adopt, not create")
	assert.Equal(t, g.ID, adopted.ID)
	assert.Len(t, store.groups, 1, "case-variant externalId create must not duplicate the group")
	members, err := p.GroupMembers(ctx, org, g.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"u1", "u2"}, members, "members must sync onto the existing row")

	// FindGroupByDisplayName (the listGroups filter probe) is case-insensitive.
	found, err := p.FindGroupByDisplayName(ctx, org, "eNgInEeRs")
	require.NoError(t, err)
	require.NotNil(t, found, "displayName filter must find the group case-insensitively")
	assert.Equal(t, g.ID, found.ID)

	// The rename-uniqueness gate rejects a case-insensitive collision: a second
	// group cannot be renamed onto an existing name that differs only in case.
	other, _, err := p.CreateGroup(ctx, org, Group{DisplayName: "Platform"})
	require.NoError(t, err)
	rename := "ENGINEERS"
	_, err = p.PatchGroup(ctx, org, other.ID, &rename, nil, nil)
	assert.ErrorIs(t, err, ErrGroupConflict, "rename onto a case-variant of an existing name must conflict")
}

// TestGroupOrgIsolation proves H6: a group created in org-a is invisible to org-b.
func TestGroupOrgIsolation(t *testing.T) {
	p, _ := newGroupSvc(t)
	ctx := context.Background()
	g, _, err := p.CreateGroup(ctx, "org-a", Group{DisplayName: "Secret"})
	require.NoError(t, err)

	_, err = p.GetGroup(ctx, "org-b", g.ID)
	assert.ErrorIs(t, err, ErrNotFound, "a cross-org group id must 404, not leak")

	_, err = p.PatchGroup(ctx, "org-b", g.ID, nil, nil, []MemberOp{{Op: "add", Members: []string{"x"}}})
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestGroupsUnavailableWithoutEngine proves group ops fail cleanly when FGA is off.
func TestGroupsUnavailableWithoutEngine(t *testing.T) {
	log := zerolog.Nop()
	store := newFakeStore()
	p := &provider{Dependencies: Dependencies{Log: &log, StorageProvider: store}} // no AuthzEngine
	_, _, err := p.CreateGroup(context.Background(), "org-a", Group{DisplayName: "X"})
	assert.ErrorIs(t, err, ErrGroupsUnavailable)
}
