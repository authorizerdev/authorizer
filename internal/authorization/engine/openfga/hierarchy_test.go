package openfga

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
)

// hierarchyModel is the canonical ReBAC hierarchy: organization → project →
// resource, where viewer/editor inherit down each edge (`X from parent`). It
// mirrors the "Org → project → resource" dashboard example. The point of the
// model is that a grant high in the tree applies to everything below it without
// a per-object tuple, while a direct grant on one object stays scoped to it.
const hierarchyModel = `model
  schema 1.1
type user
type organization
  relations
    define admin: [user]
    define editor: [user] or admin
    define viewer: [user] or editor
    define can_view: viewer
    define can_edit: editor
type project
  relations
    define org: [organization]
    define editor: [user] or editor from org
    define viewer: [user] or editor or viewer from org
    define can_view: viewer
    define can_edit: editor
type resource
  relations
    define project: [project]
    define editor: [user] or editor from project
    define viewer: [user] or editor or viewer from project
    define can_view: viewer
    define can_edit: editor`

// Subjects are referenced by their immutable user ID (user:<uuid>), never a
// display name — names are not unique and can change, IDs are stable. The IDs
// below are fixed UUIDs only so the test is deterministic.
const (
	userAlice = "user:1b9d6bcd-bbfd-4b2d-9b5d-ab8dfbbd4bed" // org-wide viewer
	userBob   = "user:2c8e7cde-ccfe-4c3e-ac6e-bc9efccd5cfe" // single-resource editor
	userCarol = "user:3d9f8def-ddff-4d4f-bd7f-cd0ffdde6dff" // no grant at all
)

// TestOpenFGAEngine_HierarchyInheritanceAndFineGrained proves the two headline
// ReBAC behaviours: (1) a single org-level grant inherits to every project and
// resource beneath it with no per-resource tuples, and (2) a fine-grained
// direct grant on one resource stays scoped to that resource.
func TestOpenFGAEngine_HierarchyInheritanceAndFineGrained(t *testing.T) {
	ctx := context.Background()
	eng, _ := newTestEngine(t)

	_, err := eng.WriteModel(ctx, hierarchyModel)
	require.NoError(t, err)

	// Structure: organization:acme → project:webapp → {resource:doc1, doc2}.
	// Grants: alice is an org viewer (once); bob is editor of doc1 only.
	err = eng.WriteTuples(ctx, []engine.TupleKey{
		// Hierarchy edges.
		{User: "organization:acme", Relation: "org", Object: "project:webapp"},
		{User: "project:webapp", Relation: "project", Object: "resource:doc1"},
		{User: "project:webapp", Relation: "project", Object: "resource:doc2"},
		// Grant ONCE on the org — no per-resource tuples for alice.
		{User: userAlice, Relation: "viewer", Object: "organization:acme"},
		// Fine-grained override — bob is editor of a single resource only.
		{User: userBob, Relation: "editor", Object: "resource:doc1"},
	})
	require.NoError(t, err)

	t.Run("org grant inherits down to every resource without a per-resource tuple", func(t *testing.T) {
		for _, obj := range []string{
			"organization:acme", "project:webapp", "resource:doc1", "resource:doc2",
		} {
			allowed, err := eng.Check(ctx, userAlice, "can_view", obj)
			require.NoError(t, err)
			assert.True(t, allowed, "org viewer must inherit can_view on %s", obj)
		}
	})

	t.Run("a viewer does not inherit edit", func(t *testing.T) {
		allowed, err := eng.Check(ctx, userAlice, "can_edit", "resource:doc1")
		require.NoError(t, err)
		assert.False(t, allowed, "viewer must not gain can_edit")
	})

	t.Run("a fine-grained grant stays scoped to its single resource", func(t *testing.T) {
		allowed, err := eng.Check(ctx, userBob, "can_edit", "resource:doc1")
		require.NoError(t, err)
		assert.True(t, allowed, "direct editor must can_edit its own resource")

		allowed, err = eng.Check(ctx, userBob, "can_edit", "resource:doc2")
		require.NoError(t, err)
		assert.False(t, allowed, "direct grant must NOT leak to a sibling resource")

		// Concentric: an editor is also a viewer of the same object.
		allowed, err = eng.Check(ctx, userBob, "can_view", "resource:doc1")
		require.NoError(t, err)
		assert.True(t, allowed, "an editor must also be able to view its resource")

		// ...but bob has no access at all to a sibling resource.
		allowed, err = eng.Check(ctx, userBob, "can_view", "resource:doc2")
		require.NoError(t, err)
		assert.False(t, allowed, "fine-grained editor has no access to doc2")
	})

	t.Run("an ungranted user is denied at every level", func(t *testing.T) {
		for _, obj := range []string{
			"organization:acme", "project:webapp", "resource:doc1",
		} {
			allowed, err := eng.Check(ctx, userCarol, "can_view", obj)
			require.NoError(t, err)
			assert.False(t, allowed, "ungranted user must be denied on %s", obj)
		}
	})

	t.Run("ListObjects enumerates every inherited resource for the org viewer", func(t *testing.T) {
		objs, err := eng.ListObjects(ctx, userAlice, "can_view", "resource")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"resource:doc1", "resource:doc2"}, objs)
	})
}
