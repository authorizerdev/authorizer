# Authorizer ReBAC guide (OpenFGA)

Authorizer's fine-grained authorization (FGA) embeds OpenFGA in-process and
models access as **relationships** between objects, not as flat per-user grants.
This guide covers the patterns that make ReBAC worth using — hierarchy and
inheritance — plus two things that are easy to get wrong: which kind of "role"
you are dealing with, and how to identify subjects.

## 1. Two kinds of "role" — they can and should differ

| | Authorizer (app) roles | FGA roles |
|---|---|---|
| What | Configured via `--roles`; carried in the JWT `roles` claim. Read in the dashboard via the admin-only `_admin_meta` query. | Relations in the model (`editor`) and `role:` objects (`role:editor`). |
| Scope | Global, coarse, identity-level — "is this principal an admin at all". | Fine-grained, object-scoped — "editor **of** `resource:301`". |
| Lives in | The token. | The authorization graph (model + tuples). |

They are **decoupled by design**. FGA roles are usually more granular than app
roles (a `viewer` *of one org*, an `editor` *of one document*), so an FGA role
name does **not** have to be one of your configured app roles. Forcing parity
would throw away ReBAC's main advantage. If you want a specific app role to be
globally assignable in the graph, *mirror it* as a tuple
(`role:admin#assignee@user:<id>`) — don't equate the two sets.

## 2. Always identify subjects by user **ID**, never by name

A tuple's subject is `user:<id>`. Use the user's **immutable id** (Authorizer's
user UUID), e.g.:

```
user:1b9d6bcd-bbfd-4b2d-9b5d-ab8dfbbd4bed   ✅ stable, unique
user:alice                                  ❌ names aren't unique and change
```

Display names and emails are not unique and can change; the user id is stable
for the lifetime of the account. The dashboard and docs use `user:<id>`
placeholders (or short ids like `user:1b9d…`); in real tuples, use the full id.
The same applies to objects: identify orgs, projects and resources by their ids
(`organization:101`), never by name. The one exception is `role:` objects,
which are keyed by the role name by design (`role:editor#assignee`).

(The engine does not hard-enforce an id format, because a subject can also be a
wildcard `user:*` or a userset like `group:9#member` / `role:editor#assignee`.
The id convention is a guideline, not a validation.)

## 3. Hierarchy: grant once, inherit everywhere

The canonical model is `organization → project → resource`, where each level
inherits viewer/editor from its parent via `X from parent`:

```dsl
model
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
    define can_edit: editor
```

The roles are **concentric** — `viewer: [user] or editor` means an editor is
automatically a viewer, so you never grant both. (This follows OpenFGA's
concentric-relationships guidance: owner ⊃ editor ⊃ viewer.)

This is the **"Org → project → resource"** example in the dashboard model
builder (Step 1 → Advanced → Browse examples).

### Wire up the structure once

```
organization:101  org      project:201     # project belongs to org
project:201     project  resource:301      # resource belongs to project
project:201     project  resource:302
```

### Grant once, high in the tree

```
user:1b9d…  viewer  organization:101          # one tuple
```

Now every check below inherits — **no per-resource tuples needed**:

```
Check(user:1b9d…, can_view, organization:101) → allowed
Check(user:1b9d…, can_view, project:201)    → allowed   (viewer from org)
Check(user:1b9d…, can_view, resource:301)     → allowed   (viewer from project ← from org)
Check(user:1b9d…, can_view, resource:302)     → allowed
```

A viewer does **not** inherit edit:

```
Check(user:1b9d…, can_edit, resource:301)     → denied
```

`ListObjects(user:1b9d…, can_view, "resource")` returns
`["resource:301", "resource:302"]` — the whole subtree, from one grant.

## 4. Fine-grained grants coexist with the hierarchy

Inheritance does not stop you from granting a single object directly. A direct
grant stays **scoped to that object**:

```
user:2c8e…  editor  resource:301              # one resource only
```

```
Check(user:2c8e…, can_edit, resource:301)     → allowed   (direct grant)
Check(user:2c8e…, can_view, resource:301)     → allowed   (concentric: editor ⊃ viewer)
Check(user:2c8e…, can_edit, resource:302)     → denied    (does NOT leak to siblings)
Check(user:2c8e…, can_view, resource:302)     → denied
```

So you compose **broad inherited access** (grant on the org/project) with
**narrow exceptions** (grant on a single resource) in the same model.

## 5. What the save paths validate

- **`_fga_write_model`** parses the DSL and runs OpenFGA's model-consistency
  check (relations reference defined types, no illegal cycles). It does **not**
  validate role names against app roles.
- **`_fga_write_tuples`** validates each tuple **against the active model** — the
  object type must exist and the relation must be defined on it, with an allowed
  user type. A tuple referencing an undefined relation/type is rejected. It does
  **not** validate `role:<x>` ids or `user:<id>` against any external list.

Both surfaces are super-admin gated and audited. The dashboard model builder
starts from a standard `admin / editor / viewer` set and offers the instance's
configured roles (read via `_admin_meta`) as optional one-click additions — FGA
roles are free to diverge from app roles (see §1).

## Where this lives in the code

- Engine: `internal/authorization/engine/openfga/` (`openfga.go` bootstrap +
  `Config`, `operations.go` model/tuple/check). SPI in
  `internal/authorization/engine/engine.go`.
- Hierarchy + fine-grained behaviour is covered by
  `internal/authorization/engine/openfga/hierarchy_test.go`.
- Every shipped model (dashboard example catalog, editor placeholder, and the
  DSL in this guide) is validated against the real embedded engine by
  `internal/authorization/engine/openfga/examples_validation_test.go` — the
  in-repo equivalent of `fga model validate`.
- Dashboard examples: `web/dashboard/src/pages/authorization/modelDsl.ts`
  (`MODEL_EXAMPLES`), tested in `modelDsl.test.ts`.
- `_admin_meta` query: `internal/graphql/admin_meta.go` (super-admin gated),
  tested in `internal/integration_tests/admin_meta_test.go`.
