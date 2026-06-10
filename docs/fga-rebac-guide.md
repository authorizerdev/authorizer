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
| Scope | Global, coarse, identity-level — "is this principal an admin at all". | Fine-grained, object-scoped — "editor **of** `resource:doc1`". |
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
for the lifetime of the account. Worked examples in the dashboard use
`user:alice` for readability only — in real tuples, use the id.

(The engine does not hard-enforce a UUID format, because a subject can also be a
wildcard `user:*` or a userset like `group:eng#member` / `role:editor#assignee`.
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
organization:acme  org      project:webapp     # project belongs to org
project:webapp     project  resource:doc1      # resource belongs to project
project:webapp     project  resource:doc2
```

### Grant once, high in the tree

```
user:1b9d…  viewer  organization:acme          # one tuple
```

Now every check below inherits — **no per-resource tuples needed**:

```
Check(user:1b9d…, can_view, organization:acme) → allowed
Check(user:1b9d…, can_view, project:webapp)    → allowed   (viewer from org)
Check(user:1b9d…, can_view, resource:doc1)     → allowed   (viewer from project ← from org)
Check(user:1b9d…, can_view, resource:doc2)     → allowed
```

A viewer does **not** inherit edit:

```
Check(user:1b9d…, can_edit, resource:doc1)     → denied
```

`ListObjects(user:1b9d…, can_view, "resource")` returns
`["resource:doc1", "resource:doc2"]` — the whole subtree, from one grant.

## 4. Fine-grained grants coexist with the hierarchy

Inheritance does not stop you from granting a single object directly. A direct
grant stays **scoped to that object**:

```
user:2c8e…  editor  resource:doc1              # one resource only
```

```
Check(user:2c8e…, can_edit, resource:doc1)     → allowed   (direct grant)
Check(user:2c8e…, can_view, resource:doc1)     → allowed   (concentric: editor ⊃ viewer)
Check(user:2c8e…, can_edit, resource:doc2)     → denied    (does NOT leak to siblings)
Check(user:2c8e…, can_view, resource:doc2)     → denied
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
seeds its roles from `_admin_meta` as a convenience, but FGA roles are free to
diverge from app roles (see §1).

## Where this lives in the code

- Engine: `internal/authorization/engine/openfga/` (`openfga.go` bootstrap +
  `Config`, `operations.go` model/tuple/check). SPI in
  `internal/authorization/engine/engine.go`.
- Hierarchy + fine-grained behaviour is covered by
  `internal/authorization/engine/openfga/hierarchy_test.go`.
- Dashboard examples: `web/dashboard/src/pages/authorization/modelDsl.ts`
  (`MODEL_EXAMPLES`), tested in `modelDsl.test.ts`.
- `_admin_meta` query: `internal/graphql/admin_meta.go` (super-admin gated),
  tested in `internal/integration_tests/admin_meta_test.go`.
