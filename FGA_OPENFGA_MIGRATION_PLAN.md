# Migration Plan: Replace Bespoke FGA with OpenFGA

**Status:** Decisions locked · Phase 0 PASSED · Phases 1–4 DONE on branch `feat/fga-engine-spi` (uncommitted) · **Precondition:** FGA is pre-stable (no GA compat guarantee) · **Type:** Breaking, full replacement

> **Implementation status:** Old FGA (Resource/Scope/Policy/Permission, #607/#610/#611) **fully removed** (never rolled out). New OpenFGA engine seam (Phase 1), GraphQL `_fga_*`/`fga_*` API + engine routed into request paths (Phase 3), and `required_relations` on session/validate (Phase 4) all **DONE** — `go build ./...` green, FGA + integration tests pass, proof-grep clean, principal-pinning + nil-engine handling tested. **Remaining:** dashboard FGA pages (Phase 5, done), SDK cleanup in the separate `authorizer-go`/`authorizer-js` repos (Phase 6), Auth0 import tool + docs (Phase 7).
>
> **Config simplification (supersedes D2 + the deployment-mode external-service framing below):** Authorizer **embeds OpenFGA in-process — it IS the engine.** The `--authorization-engine`, `--fga-mode`, and `--fga-external-url` flags were **removed**, along with three dead old-engine flags (`--authorization-cache-ttl`, `--include-permissions-in-token`, `--authorization-log-all-checks`).
>
> **FGA reuses the main database by default.** When `--database-type` is `sqlite`/`postgres`/`mysql`/`mariadb`, FGA derives its store from the existing `--database-url` automatically — **no extra flags** (OpenFGA tables live in the main DB, like the old engine; migrations run on boot, goose-locked → HA-safe). `--fga-store` (+ `--fga-store-url`) is an **override**, required only when the main DB is not OpenFGA-compatible (mongodb, dynamodb, cassandra, couchbase, arangodb, sqlserver) or to use a dedicated store. Resolved by `config.FGAStoreConfig()` (unit-tested). The driver-conflict known-issue below is **resolved** (Option B: GORM standardized on `modernc.org/sqlite`), which is what makes same-DB reuse possible.

> ✅ **RESOLVED — SQLite driver double-registration (Option B).** Linking OpenFGA's SQL datastores (`modernc.org/sqlite`) alongside Authorizer's old GORM SQLite driver (`glebarez/go-sqlite`) panicked at startup (`sql: Register called twice for driver sqlite`, since both register the name `sqlite`). Fixed by standardizing GORM on `modernc.org/sqlite` via a local dialect (`internal/storage/db/sql/sqlitedialect/`, a near-verbatim copy of glebarez's MIT dialect with the driver import swapped) and dropping `glebarez/*`. The `fga_sql` build tag was removed — OpenFGA's SQL datastores are now in the **default build**. Verified: default binary starts with no panic; embedded SQLite FGA runs in-process alongside GORM SQLite; full SQLite suite green. Pure-Go, no CGO. *(Maintenance note: the vendored dialect is now Authorizer-maintained — apply modernc/dialect updates manually.)*

> **Phase 0 spike result (FULLY validated, not assumed):** `openfga v1.17.1` embeds in-process; DSL→model→tuples→`Check`→`ListObjects` work incl. `but not` exclusion. **Persistent SQLite datastore verified** — data written by one process is read back by a separate fresh process (persistence across restart proven); bootstrap = `migrate.RunMigrations(...)` then `sqlite.New(uri, sqlcommon.NewConfig())` (pure-Go, no CGO). `go.mod` integration probe **passed** (whole authorizer tree builds with openfga added, `modernc.org/sqlite`→v1.51.0). Binary +~36–38M. See `openfga-modeling` skill for the verified bootstrap recipe + gotchas. **No open Phase 0 risks.**

---

## 0. Decisions (LOCKED — principal-engineer calls)

| # | Decision | Locked choice | Note |
|---|---|---|---|
| **D1** | Multi-DB / FGA store | **Decouple from main DB.** Embedded **SQLite = single-node/dev only**; **external Postgres/MySQL required for HA/multi-replica.** No custom datastore adapters for Mongo/Dynamo/etc. | ⚠️ SQLite cannot back multi-writer. **Honest cost: any HA install runs a second datastore** — the real price of the multi-DB moat for FGA. |
| **D2** | Deployment | **Both, configurable.** Embedded default (single-binary for dev/single-node) + **external OpenFGA first-class** (HA/distributed/cloud-native). | — |
| **D3** | Permissions API | **Replace `required_permissions` → `required_relations`.** Keep existing `roles`/`scope` filters for coarse gating; document **singleton-object pattern** (`feature:x#reader`) for capability checks. | `roles` filter already covers coarse RBAC; no dead fields kept. |
| **D4** | Roles ↔ graph | **Dual: keep `roles` claim AND mirror role grants as tuples** (`role:x#assignee@user:y`), maintained by Authorizer on role assignment. | Required for `[role#assignee]` grants + `list_objects`. |
| **Store** | Multi-tenancy | **Single global OpenFGA store; isolate tenants in the graph** (namespaced object IDs + org-membership relations). Per-store only for data-residency isolation. | OpenFGA stores aren't built for thousands of tenants. |
| **Why** | Explainability | **OpenFGA `Expand` for "why"; log decision traces for denied sensitive checks.** No custom explainer. | Compliance requirement. |

**Model-change governance:** the authorization model (DSL) is extremely powerful (one edit re-grants broadly) → admin-gated, audited, and staged (write new model version, validate, then activate).

**What is NOT removed:** core auth — login, OAuth2/OIDC, token issuance, sessions, **roles** (`roles`/`allowed_roles` claims), **OAuth scopes**, and the **custom-token-script hook**. Only the FGA Resource/Scope/Policy/Permission subsystem is removed.

> ⚠️ **Two different "scope" concepts — do not conflate.** (a) **OAuth `scope`** (the `scope` claim, `SessionQueryRequest.scope`, consent) is core OIDC — **untouched**. (b) The FGA **`Scope` entity** (resource scope) is what we delete. The removal inventory in §1 refers only to (b).

### Auth0 authz layers — what coexists with OpenFGA
OpenFGA covers only the ReBAC layer. The full Auth0-parity stack is layered; these layers are **retained and integrated**, not replaced:

| Auth0 layer | Covered by | Plan action |
|---|---|---|
| OAuth/OIDC scopes (API gate) | OAuth scopes (existing) | Keep; checked **before** FGA (cheap coarse gate) |
| RBAC roles → token | roles claims (existing) | Keep; bridge to graph per D4 |
| ReBAC (object access) | **OpenFGA (new)** | This plan |
| Actions/Rules (token pipeline) | `CustomAccessTokenScript` (existing) | Keep; expose `engine.Check`/`ListObjects` to it |
| ABAC / conditional | **OpenFGA Conditions + contextual tuples** | Fold in — no separate ABAC engine |
| Organizations / B2B | tuples (`org:X#member@user:Y`) + `org_id` claim | Model membership as tuples |
| CIBA / Token Vault / RFC 8693 delegation | — | **Out of scope** (next agent-auth layer; FGA authorizes, doesn't implement) |

**Enforcement order at runtime:** OAuth scope check → FGA `Check`. Both must pass.

---

## 1. Removal inventory (what "completely remove" deletes)

Verifiable by `grep` returning zero hits post-removal (excluding OpenFGA-new code).

**Backend — storage**
- `internal/storage/schemas/`: `resource.go`, `scope.go`, `policy.go`, `permission.go` (Resource, Scope, Policy, PolicyTarget, Permission, PermissionScope, PermissionPolicy, and the `*WithPolicies`/`*View` denorm types)
- Provider interface methods in `internal/storage/provider.go`: all `*Resource`, `*Scope`, `*Policy`, `*PolicyTarget`, `*Permission`, `*PermissionScope`, `*PermissionPolicy`, `GetPermissionsForResourceScope`
- All 6 DB implementations of the above: `db/sql/`, `db/mongodb/`, `db/arangodb/`, `db/cassandradb/`, `db/couchbase/`, `db/dynamodb/` (`resource.go`, `scope.go`, `policy.go`, `permission.go` in each)
- AutoMigrate / collection-creation entries for those schemas in each `db/*/provider.go`
- `schemas/model.go` `CollectionList` entries for those collections

**Backend — engine**
- `internal/authorization/` evaluator logic for resource/scope/policy (`evaluator.go`, parts of `cache.go`) — **repurposed**, not all deleted (the `Provider` interface + cache plumbing is reused by the new engine; see §3)

**Backend — GraphQL**
- `internal/graph/schema.graphqls`: types `AuthzResource(s)`, `AuthzScope(s)`, `AuthzPolicy/Target/Policies`, `AuthzPermission(s)`, `Permission`, `PermissionInput`; inputs `Add/Update Resource/Scope/Policy/Permission`, `PolicyTargetInput`; mutations `_authz_add/update/delete_{resource,scope,policy,permission}`; queries `_authz_{resources,scopes,policies,permissions}`, `permissions`
- `internal/graphql/`: `authz_*.go` (16 files), `permission_check.go`, `permissions.go`

**Dashboard**
- `web/dashboard/src/pages/authorization/`: `Resources.tsx`, `Scopes.tsx`, `Policies.tsx`, `Permissions.tsx`
- Authz entries in `graphql/mutation/index.ts`, `graphql/queries/index.ts`, `types.ts`

**SDKs**
- `authorizer-go`: `PermissionInput{Resource,Scope}`, `RequiredPermissions` fields on `get_session.go`, `validate_jwt_token.go`, `validate_session.go`
- `authorizer-js`: `PermissionInput`, `Permission`, `required_permissions` on session/validate types

---

## 2. Target architecture

```
Relying app / AI agent / MCP client
        │  check(user, relation, object) · list_objects · batch_check
        ▼
Authorizer GraphQL  ──►  AuthorizationEngine (OpenFGA)
   _fga_* admin              ├─ embedded openfga lib (default)   ┐
   fga_check / list          └─ external openfga (gRPC, config)  ┘
        │                                  │
   validate_jwt_token / validate_session / session
   (required_permissions → relation checks)
                                           ▼
                              FGA tuple store (SQLite | Postgres | MySQL)
```

- **Model**: OpenFGA authorization model (DSL) — types, relations, conditions.
- **Data**: relationship tuples `(object, relation, user)`.
- **Decision**: `Check`; **retrieval**: `ListObjects` (RAG pre-filter), `BatchCheck`.

### 2.1 Deployment modes (single-node / HA / serverless)
The FGA store is the deciding factor. Same engine, different backing.

| Mode | Engine | FGA store | Migrations | Notes |
|---|---|---|---|---|
| **Single-node / dev** | embedded | **SQLite file** | on boot (idempotent) OK | one process only; WAL sidecar files on local disk |
| **HA / multi-replica** | embedded *or* external | **external Postgres/MySQL** | **separate init job** | SQLite cannot back multiple writers |
| **Serverless** (Lambda/Cloud Run-scale/Vercel/Fly) | **external OpenFGA service preferred** (or embedded engine → external SQL) | **external managed Postgres/MySQL behind a connection pooler** | **separate init job; NEVER on cold start** | see rules below |

**Serverless rules (review-derived — embedded-SQLite is NOT serverless-compatible):**
- ❌ **No embedded SQLite** — ephemeral, non-shared disk; N concurrent instances can't share one file. Serverless ⇒ external SQL store, same constraint as HA but stricter.
- ❌ **No migrate-on-cold-start** — run `migrate.RunMigrations` as a deploy/init job; concurrent cold starts must not race migrations and must not pay init latency.
- ⚠️ **Connection pooling required** — per-instance pgx pools × many instances = connection explosion. Front Postgres with pgbouncer / RDS Proxy / provider pooling.
- ⚠️ **Flush before response on freeze platforms (Lambda)** — OpenFGA workers and Authorizer's fire-and-forget audit goroutine (incl. the delegation audit chain) may be frozen post-response; ensure audit writes complete before returning, or enqueue.
- ✅ **External `memory_store`** (Redis/DB) already serverless-ready — used for sessions, the FGA decision cache (embedded mode only), and the delegation revocation list.
- ✅ **Don't cache FGA decisions in external mode** (locked rule) — so no stale-allow across ephemeral instances.
- **Platform fit:** Cloud Run/Fly (process alive while warm) tolerate the embedded engine + external store; Lambda/Vercel (freeze-based) prefer the **external OpenFGA service** so the function binary stays lean and no in-process engine state depends on the frozen runtime.

---

## 3. Phased plan (goal-driven; each phase has a verify gate)

### Phase 1 — Engine seam + OpenFGA embed
1. Add `internal/authorization/engine` interface: `Check(ctx, user, relation, object, ctxTuples) (bool, error)`, `ListObjects(ctx, user, relation, type) ([]string, error)`, `BatchCheck`, `WriteTuples`, `DeleteTuples`, `ReadTuples`, `WriteModel`, `ReadModel`.
2. Vendor `github.com/openfga/openfga` as an in-process library; implement `engine.openfga`.
3. Wire FGA store config: `--fga-store=sqlite|postgres|mysql`, `--fga-store-url`, `--fga-mode=embedded|external`, `--fga-external-url`.
4. Init in `cmd/root.go` after Storage/MemoryStore.

→ **Verify:** unit test writes a model + tuples, `Check` returns expected allow/deny against the embedded store; server boots with `--fga-mode=embedded` on a Mongo main DB.

### Phase 2 — Remove bespoke storage + engine *(deferred one release — review fix #4)*
**Do not big-bang.** Ship Phase 1's SPI with **both** engines for one release: OpenFGA default, old `policy` engine still selectable (`--authorization-engine=fga|policy`). Validate FGA in production, *then* remove the old engine in the following release. The "completely remove" directive still holds — just one release later, behind the seam that exists precisely to de-risk this.
1. (Release N) Both engines live behind SPI; default `fga`.
2. (Release N+1) Delete schemas, provider methods, and all 6 DB impls per §1; remove AutoMigrate / collection entries; delete dead `evaluator.go` resource/scope/policy paths (keep cache plumbing reused by the new engine).

→ **Verify:** N: both engines pass integration tests. N+1: `grep -r "Resource\|Scope\|Policy\|Permission" internal/storage` returns only unrelated hits; `go build ./...` green; `make test-all-db` green.

### Phase 3 — GraphQL API replacement (breaking)
1. Remove all `_authz_*` + `permissions` + `Permission*` schema/resolvers.
2. Add admin model/tuple management (admin-gated, `_` prefix):
   - `_fga_write_model(params): FgaModel!` / `_fga_get_model: FgaModel!`
   - `_fga_write_tuples(params: [FgaTupleInput!]!): Response!`
   - `_fga_delete_tuples(params: [FgaTupleInput!]!): Response!`
   - `_fga_read_tuples(params: FgaReadInput!): FgaTuples!`
3. Add runtime check API (authenticated, principal = `user:<id>`):
   - `fga_check(params: FgaCheckInput!): FgaCheckResponse!`
   - `fga_list_objects(params: FgaListObjectsInput!): FgaObjects!`
   - `fga_batch_check(params: [FgaCheckInput!]!): [FgaCheckResponse!]!`
   - `FgaCheckInput { object: String!, relation: String!, context: Map }`
4. `make generate-graphql`.

→ **Verify:** integration test: admin writes model+tuples via GraphQL, authenticated user gets correct `fga_check`/`fga_list_objects` results; old `_authz_*` ops return "unknown field".

### Phase 4 — session / validate_session / validate_jwt (the API the user flagged) *(D3 — review fix #2)*
**Coarse vs fine are different questions — don't force capabilities into ReBAC.**
- **Coarse gating stays** on the existing `roles` (and `scope`) filters — pure "user must have role/scope X" needs no graph.
- **Fine gating is new:** replace `required_permissions` with `required_relations: [{object, relation}]` (AND, OpenFGA `Check` with `user:<principal.ID>`).
- **Capability-style checks** (e.g., "can read reports at all") use the **singleton-object pattern** — model `feature:reports#reader` and check `{object: "feature:reports", relation: "reader"}`. Documented, not awkward.
1. Schema: remove `required_permissions: [PermissionInput!]`; add `required_relations: [FgaRelationCheck!]` on `SessionQueryRequest`, `ValidateJWTTokenRequest`, `ValidateSessionRequest`. Keep `roles`/`scope` filters untouched.
2. Replace `enforceRequiredPermissions` → `enforceRequiredRelations`: loop `engine.Check(user, rel.relation, rel.object)`, AND semantics, fail-closed, keep metrics/labels shape.
3. Update `session.go`, `validate_jwt_token.go`, `validate_session.go` call sites.

→ **Verify:** `validate_session` with a satisfied relation → authorized; unsatisfied → `unauthorized`; empty list still authorizes; existing `roles` filter still gates coarsely.

### Phase 5 — Dashboard
Replace `pages/authorization/{Resources,Scopes,Policies,Permissions}.tsx` with:
1. **Authorization Model** page — DSL editor (textarea + validate-on-save via `_fga_write_model`), shows current model + version.
2. **Relationship Tuples** page — table + add/delete (`object`, `relation`, `user`), backed by `_fga_read/write/delete_tuples`.
3. **Access Tester** page — form (`user`, `relation`, `object`) → calls `fga_check`, shows allow/deny + (optionally) `expand`.
4. Update `graphql/mutation|queries/index.ts`, `types.ts`, route stays `authorization/*`.

→ **Verify:** `make build-dashboard` green; manual: define model, add tuple, tester returns allow; remove tuple, tester returns deny.

### Phase 6 — SDKs (client-facing surface only, per agreed scope — no admin CRUD)
**authorizer-go** and **authorizer-js**:
1. Remove `PermissionInput`/`Permission`/`required_permissions`.
2. Add `required_relations` to `GetSession`/`ValidateJWTToken`/`ValidateSession` params.
3. Add client methods: `FgaCheck`, `FgaListObjects`, `FgaBatchCheck` (+ a thin `FgaRetriever`-style helper for RAG pre-filtering in each).

→ **Verify:** SDK unit tests against a running server: `FgaCheck` allow/deny correct; `FgaListObjects` returns expected IDs; `ValidateSession` honors `required_relations`.

### Phase 7 — Auth0 import tool + Docs *(review fix #7 — the migration value, actually built)*
1. **Auth0 FGA → Authorizer import** CLI/endpoint: ingest an Auth0/OpenFGA **model (DSL)** and **tuple export** → write via `_fga_write_model` / `_fga_write_tuples`. This is the headline migration deliverable; without it "ports 1:1" is just a claim.
2. `MIGRATION.md`: breaking-change notes + Auth0-FGA→Authorizer mapping; singleton-object pattern guide for coarse checks.
3. Document `--fga-*` flags, **SQL-store requirement (single-node SQLite vs HA external Postgres/MySQL — D1)**, embedded vs external.
4. Update `ROADMAP_V2.md`: FGA = OpenFGA ReBAC.

→ **Verify:** a real Auth0 FGA export imports and `fga_check` reproduces the same decisions; a follower can stand up FGA from the guide alone.

---

## 4. Cross-cutting

- **Audit/metrics:** mirror existing `RecordAuthzCheck` / required-permissions metric shape for the new check + relation paths (low-cardinality labels).
- **Cache (review fix #5 — don't double-cache):** **only cache in embedded mode**, where Authorizer intercepts every tuple/model write and can invalidate; key on `(user, relation, object, model_version)`. In **external mode, do NOT layer Authorizer's cache** — tuples can be written directly to OpenFGA, so rely on OpenFGA's own consistency/caching. A second cache there = stale-allow bug.
- **`list_objects` cost (review fix #9):** expensive + an enumeration surface → mandatory pagination, result cap, latency budget, and rate-limit/DoS guards on `fga_list_objects`.
- **Security:** `_fga_*` admin-gated (`IsSuperAdmin`); `fga_*` authenticated, principal pinned to token `sub` (never client-supplied user); fail-closed on engine error. **Model edits** admin-gated + audited + staged (write→validate→activate).
- **Test matrix:** `make test-all-db` must pass to prove removal left no DB-impl dangling refs, even though the FGA store itself is SQL-only.

## 5. Ordering / rollout *(review fix #4 — seam, not big-bang)*
- **Release N:** Phase 1 (SPI + embed) + Phases 3/4 (new API) ship with **both engines** behind `--authorization-engine`, default `fga`. Old engine still selectable.
- **Release N+1:** Phase 2 removal once FGA is validated in production.
- **Then:** 5 (dashboard) + 6 (SDKs) once the API is frozen; 7 (import tool + docs).
- **Prereqs (review fix #10):** Wave-2 delegation depends on agent identity + M2M/client-credentials (roadmap Phase 2) and OAuth 2.1 AS / DCR (Phase 4.1) — not started here.

## 6. Open risks
1. **D1 multi-DB** — the central tradeoff (above).
2. **Embedding maturity** — confirm `openfga` embeds cleanly as a lib (Phase 0 spike) vs. forcing external mode.
3. **Consistency window** — document OpenFGA consistency options for distributed deployments.
4. **SDK scope** — admin tuple CRUD intentionally excluded from SDKs (matches agreed client-facing-only scope).
