package http_handlers

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	fgaengine "github.com/authorizerdev/authorizer/internal/authorization/engine/openfga"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// newRoleTestEngine spins up an in-memory FGA engine loaded with groupTestModel
// (defined in saml_idp_groups_test.go), whose `role` type already accepts both
// direct user assignees and nested group#member usersets.
func newRoleTestEngine(t *testing.T, storeName string) engine.AuthorizationEngine {
	t.Helper()
	logger := zerolog.New(zerolog.NewTestWriter(t))
	eng, err := fgaengine.New(
		&fgaengine.Config{Store: fgaengine.StoreMemory, StoreName: storeName},
		&fgaengine.Dependencies{Log: &logger},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c, ok := eng.(interface{ Close() }); ok {
			c.Close()
		}
	})
	_, err = eng.WriteModel(context.Background(), groupTestModel)
	require.NoError(t, err)
	return eng
}

// finalClaimRoles reproduces exactly what the org-scoped mint points
// (issueSSOSession / issueSAMLSession) put into AuthTokenConfig.Roles:
// splitRoles(user.Roles) unioned with the org's FGA-derived roles.
func finalClaimRoles(h *httpProvider, orgID string, user *schemas.User, log *zerolog.Logger) []string {
	return unionRoles(splitRoles(user.Roles), h.orgGroupDerivedRoles(context.Background(), orgID, user, log))
}

// TestOrgGroupDerivedRolesHappyPath: a user who is a member of a group bound to
// role "admin" in Org A gets "admin" in the roles claim minted for Org A —
// through both the transitive group→role path and a direct user→role grant.
func TestOrgGroupDerivedRolesHappyPath(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	ctx := context.Background()
	eng := newRoleTestEngine(t, "authorizer-org-roles-happy")

	const (
		orgA    = "org-a"
		userID  = "user-1"
		groupID = "gid-a"
	)
	// user -> group (SCIM membership), group -> role:admin (admin-written binding).
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + userID, Relation: "member", Object: "group:" + orgA + "/" + groupID},
		{User: "group:" + orgA + "/" + groupID + "#member", Relation: "assignee", Object: "role:" + orgA + "/admin"},
		// A direct user->role grant in the same org must also project.
		{User: "user:" + userID, Relation: "assignee", Object: "role:" + orgA + "/editor"},
	}))

	h := &httpProvider{Dependencies: Dependencies{Log: &logger, AuthzEngine: eng}}
	user := &schemas.User{ID: userID, Roles: "user"} // global default role

	derived := h.orgGroupDerivedRoles(ctx, orgA, user, &logger)
	assert.ElementsMatch(t, []string{"admin", "editor"}, derived)

	// The role lands in the JWT claim, and the pre-existing global role survives.
	claim := finalClaimRoles(h, orgA, user, &logger)
	assert.Equal(t, "user", claim[0], "existing role must stay first / never be dropped")
	assert.ElementsMatch(t, []string{"user", "admin", "editor"}, claim)
}

// TestOrgGroupDerivedRolesCrossTenantContainment is the security-critical proof:
// a user who legitimately holds the "admin" role in Org B must NEVER have that
// role appear in a token minted for Org A.
func TestOrgGroupDerivedRolesCrossTenantContainment(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	ctx := context.Background()
	eng := newRoleTestEngine(t, "authorizer-org-roles-xtenant")

	const (
		orgA   = "org-a"
		orgB   = "org-b"
		userID = "user-shared" // member of a group in BOTH orgs
	)
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		// Org A: viewer role via group.
		{User: "user:" + userID, Relation: "member", Object: "group:" + orgA + "/ga"},
		{User: "group:" + orgA + "/ga#member", Relation: "assignee", Object: "role:" + orgA + "/viewer"},
		// Org B: admin role via group (the dangerous one).
		{User: "user:" + userID, Relation: "member", Object: "group:" + orgB + "/gb"},
		{User: "group:" + orgB + "/gb#member", Relation: "assignee", Object: "role:" + orgB + "/admin"},
	}))

	h := &httpProvider{Dependencies: Dependencies{Log: &logger, AuthzEngine: eng}}
	user := &schemas.User{ID: userID, Roles: "user"}

	// Token for Org A carries ONLY org-A's viewer, NEVER org-B's admin.
	claimA := finalClaimRoles(h, orgA, user, &logger)
	assert.ElementsMatch(t, []string{"user", "viewer"}, claimA)
	assert.NotContains(t, claimA, "admin", "cross-tenant leak: org-B role surfaced in an org-A token")

	// Symmetric: token for Org B carries ONLY org-B's admin.
	claimB := finalClaimRoles(h, orgB, user, &logger)
	assert.ElementsMatch(t, []string{"user", "admin"}, claimB)
	assert.NotContains(t, claimB, "viewer")
}

// TestOrgGroupDerivedRolesGate2RejectsSlash: an object id that slips the Gate 1
// prefix but is not a bare role name (contains a further "/") is rejected by the
// shape gate — never emitted as a role.
func TestOrgGroupDerivedRolesGate2RejectsSlash(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	ctx := context.Background()
	eng := newRoleTestEngine(t, "authorizer-org-roles-gate2")

	const orgA = "org-a"
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:u", Relation: "assignee", Object: "role:" + orgA + "/nested/evil"},
	}))

	h := &httpProvider{Dependencies: Dependencies{Log: &logger, AuthzEngine: eng}}
	got := h.orgGroupDerivedRoles(ctx, orgA, &schemas.User{ID: "u"}, &logger)
	assert.Empty(t, got, "gate 2 must reject a non-bare role name")
}

// TestOrgGroupDerivedRolesRegression proves the non-FGA-group case is unchanged:
// a user with no role bindings derives nothing, so the claim is exactly what it
// was before this change (splitRoles(user.Roles)).
func TestOrgGroupDerivedRolesRegression(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	ctx := context.Background()
	eng := newRoleTestEngine(t, "authorizer-org-roles-regression")

	h := &httpProvider{Dependencies: Dependencies{Log: &logger, AuthzEngine: eng}}
	user := &schemas.User{ID: "lonely", Roles: "user,cashier"}

	assert.Empty(t, h.orgGroupDerivedRoles(ctx, "org-a", user, &logger))
	// Claim is byte-for-byte the pre-change behaviour.
	assert.Equal(t, splitRoles(user.Roles), finalClaimRoles(h, "org-a", user, &logger))
}

// TestOrgGroupDerivedRolesFailClosed: with no engine configured, and on any
// lookup error, no roles are derived — never an error, never a partial set.
func TestOrgGroupDerivedRolesFailClosed(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	ctx := context.Background()

	// No engine at all.
	hNoEngine := &httpProvider{Dependencies: Dependencies{Log: &logger}}
	assert.Nil(t, hNoEngine.orgGroupDerivedRoles(ctx, "org-a", &schemas.User{ID: "u"}, &logger))

	// Engine present but no model written yet → ListObjects errors → fail-closed.
	logger2 := zerolog.New(zerolog.NewTestWriter(t))
	engNoModel, err := fgaengine.New(
		&fgaengine.Config{Store: fgaengine.StoreMemory, StoreName: "authorizer-org-roles-nomodel"},
		&fgaengine.Dependencies{Log: &logger2},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c, ok := engNoModel.(interface{ Close() }); ok {
			c.Close()
		}
	})
	hNoModel := &httpProvider{Dependencies: Dependencies{Log: &logger, AuthzEngine: engNoModel}}
	assert.Nil(t, hNoModel.orgGroupDerivedRoles(ctx, "org-a", &schemas.User{ID: "u"}, &logger))
}

// TestUnionRoles covers the additive union helper: dedup, order preservation,
// and the never-fewer-than-base guarantee.
func TestUnionRoles(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, unionRoles([]string{"a", "b"}, nil))
	assert.Equal(t, []string{"a", "b", "c"}, unionRoles([]string{"a", "b"}, []string{"c"}))
	// Duplicates in extra are dropped; base order preserved; base is a prefix.
	assert.Equal(t, []string{"a", "b", "c"}, unionRoles([]string{"a", "b"}, []string{"a", "c", "b"}))
	// Empty strings in extra are ignored.
	assert.Equal(t, []string{"a"}, unionRoles([]string{"a"}, []string{"", "a"}))
}
