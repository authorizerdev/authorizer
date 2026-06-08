# Enterprise Authorization Model (OpenFGA ReBAC)

How an enterprise expresses **fine-grained permissions on resources by role**, plus **one-off user-specific grants**, plus **user-specific exceptions/denials** — and how all of them mix in a single check.

---

## The mental model: every check is `UNION of grants − exclusions`

Four ways a user gets (or loses) access, all evaluated together:

| Layer | Mechanism | Example |
|---|---|---|
| **1. Role-based** (broad) | Assign a *role* to a relation → everyone with the role inherits | "all `editor`s can edit project docs" |
| **2. Structural** (hierarchy) | Membership + inheritance org→team→project→doc | "engineering team members can view phoenix docs" |
| **3. User-specific grant** (additive) | Direct tuple to one user | "Frank gets view on this one budget" |
| **4. User-specific exception** (subtractive) | `blocked` relation + `but not` | "Erin is blocked from this sensitive doc despite team membership" |

A `Check` resolves to: **(role grants ∪ structural grants ∪ direct grants) − blocked**. That intersection/union/exclusion *is* "all the mix-matches."

---

## The authorization model (DSL — authored once)

```dsl
model
  schema 1.1

type user

# Roles are first-class objects so they can be assigned to any relation (RBAC-in-ReBAC)
type role
  relations
    define assignee: [user]

type organization
  relations
    define admin:  [user, role#assignee]
    define member: [user, role#assignee] or admin

type team
  relations
    define org:    [organization]
    define member: [user, team#member] or admin from org   # org admins are implicitly team members

type project
  relations
    define team:   [team]
    define lead:   [user, role#assignee]
    define editor: [user, role#assignee] or lead or member from team
    define viewer: [user, role#assignee] or editor

type document
  relations
    define project: [project]
    define owner:   [user]
    define blocked: [user]                                  # user-specific exception
    define editor:  [user, role#assignee] or owner or editor from project
    define viewer:  [user, role#assignee] or editor or viewer from project
    define can_edit: editor but not blocked                 # effective permission
    define can_view: viewer but not blocked
```

Key constructs that deliver each layer:
- `[role#assignee]` → **role-based** grants (assign a whole role to a relation).
- `[user]` → **user-specific** one-off grants.
- `X from Y` → **hierarchical inheritance** (e.g., `editor from project`).
- `but not blocked` → **user-specific exception/deny** override.

---

## Worked scenario: Acme Corp

**Structure:** `org:acme` → `team:engineering`, `team:finance` → `project:phoenix` (eng), `project:ledger` (finance) → `doc:design-spec` (phoenix), `doc:q4-budget` (ledger).
**Roles:** `role:org-admin`, `role:editor`, `role:auditor`.

### Relationship tuples (the data — written at runtime)

```text
# --- roles & org ---
role:org-admin#assignee        @ user:alice
organization:acme#admin        @ role:org-admin#assignee     # role grants org admin
team:engineering#org           @ organization:acme
team:finance#org               @ organization:acme

# --- structural membership ---
team:engineering#member        @ user:bob
team:engineering#member        @ user:erin
team:finance#member            @ user:carol

# --- project wiring + role-based grant ---
project:phoenix#team           @ team:engineering
project:ledger#team            @ team:finance
project:phoenix#editor         @ role:editor#assignee        # role:editor → edit phoenix
role:editor#assignee           @ user:bob
project:ledger#viewer          @ role:auditor#assignee       # role:auditor → view ledger
role:auditor#assignee          @ user:dave

# --- documents ---
document:design-spec#project   @ project:phoenix
document:q4-budget#project     @ project:ledger

# --- user-specific overrides ---
document:design-spec#blocked   @ user:erin                   # exception: deny (layer 4)
document:q4-budget#viewer      @ user:frank                  # odd one-off grant (layer 3)
```

### Resulting access (Check results)

| User | Why | `can_view design-spec` | `can_edit design-spec` | `can_view q4-budget` |
|---|---|---|---|---|
| **alice** | org-admin role → admin → (member→editor→) inherits everywhere | ✅ | ✅ | ✅ |
| **bob** | `role:editor` → project editor → doc editor | ✅ | ✅ | ❌ |
| **erin** | eng member → would inherit viewer **but blocked** | ❌ *(exception)* | ❌ *(exception)* | ❌ |
| **frank** | not a member; **direct one-off** viewer on budget only | ❌ | ❌ | ✅ *(one-off)* |
| **dave** | `role:auditor` → view ledger only (read) | ❌ | ❌ | ✅ *(role, read-only)* |
| **carol** | finance member; no path to eng docs | ❌ | ❌ | ✅ |

### The mix-match, on one resource

`doc:design-spec` effective **can_edit** set =

```
  { alice }            # role-based (org-admin) + structural inheritance
∪ { bob }              # role-based (role:editor on project)
∪ { <project leads> }  # role/structural
∪ { <direct owners> }  # user-specific grant
−  { erin }            # user-specific exception (but not blocked)
```

One check, four layers, resolved by the engine. Adding a contractor for a day = one tuple; revoking = delete it. No role explosion, no policy rewrite.

---

## ABAC / conditional twist (optional)

Need "editors may edit **only during business hours**" or "**only from corp network**"? Attach an OpenFGA **Condition**:

```dsl
condition in_business_hours(now: timestamp) {
  now.Hours >= 9 && now.Hours < 18
}
type document
  relations
    define editor: [user, role#assignee with in_business_hours] or ...
```

Context (`now`, IP, purpose) is passed at check time — no separate ABAC engine.

---

## How the enterprise drives this (Dashboard + API)

| Task | Dashboard | API |
|---|---|---|
| Define the model | **Authorization Model** page (DSL editor) | `_fga_write_model` |
| Assign a role to a user | "Assign role" (writes `role:X#assignee@user:Y`) | `_fga_write_tuples` |
| Grant role broad access on a resource type | Model relation `[role#assignee]` + tuple | `_fga_write_tuples` |
| One-off share to a user | "Share resource" | `_fga_write_tuples` |
| Block/except a user | "Block user" (writes `#blocked`) | `_fga_write_tuples` |
| Ask "can X do Y on Z?" | **Access Tester** | `fga_check` |
| "What can X see?" (UI / RAG) | — | `fga_list_objects` |
| "Who can see Z?" | resource access panel | `expand` |

---

## Why this is the enterprise sweet spot

- **Roles still feel like roles** (assign a role, broad access follows) — admins keep the mental model they know.
- **One-off exceptions don't break the model** — additive grants and `but not` exclusions are just tuples.
- **Hierarchy is automatic** — grant at org/team/project, resources inherit.
- **Reverse queries** (`list_objects`) power both AI retrieval and "show me everything I can access" UIs.
- **Auditable & revocable** — every grant is a tuple with a clear provenance; revoke = delete.
```
