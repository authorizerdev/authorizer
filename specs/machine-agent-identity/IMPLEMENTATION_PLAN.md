# Machine & Agent Identity — Implementation Plan

Branch: `feat/machine-agent-identity` (off `origin/main`). Single large branch, manually tested per phase before any merge.

Design source of truth: the Rev. 5.1 design doc (client registry, unified `TrustedIssuer`, organizations, SSO/SCIM, delegation). Section references below (§4.1 etc.) point at that doc.

This plan is written so an implementing subagent can follow it without re-deriving decisions. Each feature lists **DB · API · UI · Tests** and the **build gate** that must pass before the next unit starts.

---

## Ground rules (every subagent obeys)

1. **Never commit to `main`.** Work only on `feat/machine-agent-identity`.
2. **Schema change ⇒ all 6 storage providers.** SQL (GORM), MongoDB, ArangoDB, Cassandra/Scylla, DynamoDB, Couchbase. Wire the table into `internal/storage/schemas/model.go` `Collections`, then each provider's migration path.
3. **`make generate-graphql`** after editing `internal/graph/schema.graphqls`; commit `gen/`-equivalent output.
4. **`make proto-gen`** after editing `proto/`; commit `gen/`.
5. **Transports stay thin.** Shared logic in `internal/service/` behind a `Provider` interface (`Dependencies` struct + `New()`), not in handlers.
6. **Admin GraphQL ops are `_`-prefixed.**
7. **Secrets carry `json:"-"`** and never appear in a read projection. Provider round-trip test must assert the hash persists AND never serializes out.
8. **Build gate = `go build ./... && make lint-go && go vet ./...`** compiles clean; **test gate = `TEST_DBS=sqlite go test -p 1 ./internal/...`** green. Storage tests honor `TEST_DBS`.
9. **gin-context error pattern** (PR #638/#639): resolvers do `gc, err := utils.GinContextFromContext(ctx)`, return the error, record the security metric; fire-and-forget goroutines use `context.WithoutCancel`.
10. **Audit events** use the dotted `<actor>.<action>` scheme in `internal/constants/audit_event.go`, success AND failure.

---

## Phase A — Client registry foundation (the load-bearing phase)

Goal: replace the global `Config.ClientID`/`ClientSecret` and the (unreleased) `ServiceAccount` entity with one authoritative `authorizer_clients` table, behind a shared client-authentication resolver, **without breaking existing single-client deployments**.

### A0 — Backward-compat invariants (write these as tests FIRST, then keep them green)

- **BC1**: the seeded reserved client's OAuth `client_id` == the literal `Config.ClientID`. `aud` on every token, JWKS `kid`, introspection `client_id`, and `client_check` all continue to key on that exact string. A surrogate UUID may only be an internal PK.
- **BC2**: the session-cookie AES key stays bound to `Config.ClientSecret` (or a dedicated key), NOT the per-client bcrypt hash. `crypto.EncryptAES(Config.ClientSecret, …)` in `auth_token.go` is untouched; rotating a client's stored secret must not change cookie crypto.
- **BC3**: multi-client `aud` minting/validation is explicitly in this phase — `jwt.go` single-`aud` validation and introspection audience check become registry-aware, but the existing client's `aud` stays `Config.ClientID`.

Regression tests (add before code): existing token still validates; existing session cookie still decrypts; discovery/JWKS shapes unchanged; introspection response fields unchanged.

### A1 — Schema: `authorizer_clients`

- **DB**: struct `internal/storage/schemas/client.go` (see design §4.7 table). Columns: `id` (PK surrogate), `client_id` (unique), `kind` (immutable: `interactive`|`service_account`), `client_secret_hash` (`json:"-"`, nullable), `redirect_uris`, `grant_types`, `allowed_scopes`, `max_scopes`, `token_endpoint_auth_method`, `org_id` (nullable FK), timestamps. Add to `Collections`. Migrate in all 6 providers. Index `client_id` (unique) and `org_id`.
- **Tests**: shared `internal/storage/provider_test.go` round-trip on the struct — write→read→assert every field incl. `client_secret_hash` persists and never JSON-serializes; unique `client_id`; `org_id` filter. Runs on all `TEST_DBS`.
- **Build gate**, then **test gate** on sqlite.

### A2 — Seed + config bridge

- **DB**: idempotent upsert keyed on `client_id == Config.ClientID` at boot, after `storage.New` in `cmd/root.go` (before Token/HTTP). Seed as `kind=interactive`, confidential, PKCE enforced, `client_secret_hash = bcrypt(Config.ClientSecret)`. Idempotent (skip-if-present); read-replica/no-write-path instance skips rather than fatals.
- **Tests**: boot twice → one row; concurrent seed → no duplicate (per-provider upsert semantics).

### A3 — Client-authentication resolver

- **API/service**: new `internal/service/clientauth` (or fold into an existing service) `Provider` with `ResolveAndAuthenticate(ctx, req) (*Client, error)`. Methods: `client_secret_basic`, `client_secret_post`, `none` (public+PKCE). RFC 6749 §2.3: reject >1 auth method. Constant-time / dummy-hash on miss for every kind (extend the existing service-account timing guard to the interactive path). Cached read-through lookup in the shared `memory_store` (Redis), cross-instance invalidation on write; `Config.ClientID` remains bootstrap seed + fallback so an empty/unreachable table can't lock out login.
- Repoint the 6+ static config-comparison sites to consume the resolver: `token.go`, `introspect.go` (caller-auth per RFC 7662), `revoke_refresh_token.go` (token-ownership per RFC 7009), `client_check.go`, `app.go` (serve reserved client's id), `authorize.go`.
- **Tests**: each method authenticates the seeded client; wrong secret rejected constant-time; dual-method rejected; unknown client → dummy-hash timing; introspection/revoke ownership checks.

### A4 — GraphQL/gRPC admin surface `_client_*`

- **API**: `internal/graph/schema.graphqls` — `type Client`, `_create_client`/`_update_client`/`_delete_client`/`_client`/`_clients` (`_`-prefixed). Ship these names directly; never ship `_service_account_*`. `kind` immutable in `_update`. Kind-aware projection (never returns `client_secret_hash`). One-time secret reveal on create for `service_account`/confidential. Admin-suppliable `client_id` (default system-generated). `make generate-graphql`. Mirror on gRPC admin (`internal/grpcsrv/handlers/`).
- **UI**: `web/dashboard` — a "Clients" admin page (list/create/edit/delete, one-time secret reveal). Keep `meta.client_id` returning the reserved client. `web/app` `/app` bootstrap stays pinned to the reserved client (defer 2nd-interactive-client delivery).
- **Tests**: integration test — create client via GraphQL, secret revealed once, never leaks on get/list; `kind` immutable; admin-supplied id honored.

### A5 — service_account grant + discovery metadata + refresh rotation

- **API**: `client_credentials` grant on `/oauth/token` uses the resolver; `service_account` client, `max_scopes` ceiling, scope-subset enforcement, bcrypt-12 secret, audit `token.client_credentials`/`_failed`. `.well-known/openid-configuration` + `oauth-authorization-server` advertise `grant_types_supported` incl. `client_credentials` and (reserved for B/D) the new methods; refresh-token rotation for public interactive clients (RFC 9700 §4.14.2); exact redirect-URI matching per client (§4.1/G10).
- **Tests**: e2e — admin creates `service_account`, authenticates at `/oauth/token`, token validates, scopes enforced; discovery advertises `client_credentials`; refresh rotation detects replay.

**Phase A done when**: all gates green on sqlite + the backward-compat regression suite passes + a manual smoke of the existing login flow works unchanged.

---

## Phase B — Secretless client auth (`TrustedIssuer` + client_assertion)

### B1 — Schema: `authorizer_trusted_issuers` (unified)

- **DB**: struct with `kind` (`client_assertion_trust`|`sso_saml`|`sso_oidc`, immutable), `org_id` (nullable, immutable, FK), `client_id` (nullable FK), `issuer_url`, `audience`, `expected_subject`, `jwks_url`/`discovery_url`/`pinned_jwks`, `config` (kv map). Only `client_assertion_trust` populated in B; `sso_*` reserved for C. All 6 providers + `Collections`. Index `(kind, org_id, issuer_url)`.
- **Tests**: round-trip ×6; `kind`/`org_id` immutability guard; `expected_subject` empty ⇒ deny-all.

### B2 — client_assertion resolution (RFC 7523)

- **API**: extend the A3 resolver with `client_assertion` (`client_assertion_type=…jwt-bearer` and `…jwt-spiffe`). **Lookup scoped `WHERE kind='client_assertion_trust' AND org_id IS NULL`** (CR1); reject any other row. Validate: signature against JWKS keyed by trust-row identity (H7); `aud` == exact token-endpoint URL; `exp − iat ≤ ceiling`; single-use `jti` in shared store, or `(sub,iat,exp)` replay key when `jti` absent (H4); subject exact-match by default, anchored patterns only (H3); algorithm allow-list, reject `alg:none` (G8). SSRF hardening on org-supplied URLs already in place from design; issuer URLs globally unique, reject `kubernetes.default.svc` (H5). SPIFFE: validate SPIFFE ID against trust bundle per `draft-ietf-oauth-spiffe-client-auth`.
- **API**: advertise `private_key_jwt` + `…jwt-spiffe` in discovery `token_endpoint_auth_methods_supported`.
- **Tests**: K8s projected-token happy path; replay rejected (jti and no-jti); wrong subject rejected; SSO row rejected on this path (CR1); `alg:none` rejected; cross-cluster issuer collision rejected.

### B3 — Admin surface + UI for trusted issuers

- **API**: `_create_trusted_issuer` etc. (platform-super-admin only for `client_assertion_trust`). **UI**: dashboard "Trusted Issuers" page.
- **Tests**: only super-admin can create `client_assertion_trust`.

---

## Phase C — Organizations, SSO & provisioning

**Hard gate**: FGA `HIGHER_CONSISTENCY` must be plumbed first (MED-1) — add an optional consistency param to `internal/authorization/engine` `CheckRequest`, used for grant checks at token issuance. And an **org-scoped admin permission model** (H1) must exist before org operations.

### C1 — Schema: `authorizer_organizations`, `authorizer_org_memberships`, `authorizer_scim_endpoints`

- **DB**: three tables (design §4.7) ×6 providers + `Collections`. `users` additive: `external_id` (nullable), `is_active` (default true). Unique `(org_id, user_id)` on memberships; unique `org_id` on scim_endpoints.
- **Tests**: round-trip ×6; membership uniqueness; user additive fields default correctly for existing rows.

### C2 — `(org, client)` grant as FGA tuple + org_id claim

- **API**: grant create requires bidirectional consent (org admin + client owner). `org_id` claim stamped only from an active, consistency-checked grant tuple (`org:<id>#client_grant@client:<id>`), never a request param (C3). `_grant_org_client` / `_revoke_org_client` admin ops.
- **Tests**: token for un-granted org rejected; revoked grant immediately stops stamping (needs HIGHER_CONSISTENCY); rogue-org-admin cannot pull in a client without owner consent.

### C3 — SSO connections (SAML SP + OIDC) on `trusted_issuers`

- **API**: `sso_saml`/`sso_oidc` rows (org-scoped). SAML SP with the full §4.4 invariants (signature over consumed assertion / XSW defense, unsigned rejected, per-org Audience+Recipient+Destination, NotBefore/NotOnOrAfter+skew, single-use AssertionID cache in shared store, InResponseTo bound to pending AuthnRequest store, IdP-initiated off by default, RelayState allow-listed). OIDC broker with mix-up defense via `iss`/RFC 9207 (G3). Federated identity namespaced `(org_id, issuer, sub)`, no email-only linking. JIT provisioning.
- **UI**: dashboard per-org connection config (metadata, cert, ACS URL display).
- **Tests**: valid assertion → session; XSW/unsigned/wrong-audience/replay/cross-org all rejected; mix-up rejected.

### C4 — Inbound SCIM 2.0 server (per org)

- **API**: router group `/scim/v2/…`, SCIM error schema. Endpoints: Users (POST/GET/PUT/PATCH/DELETE-as-active=false), Groups, `/ServiceProviderConfig`, `/Schemas`, `/ResourceTypes`. `userName eq` filtering. `org_id` derived only from the connection bearer token (H6). Group→role mapping bounded to org namespace, unknown groups ignored (CR2). Deactivate/delete synchronously revokes sessions + refresh tokens. Bearer token high-entropy, hashed at rest, constant-time.
- **UI**: dashboard shows SCIM endpoint URL + rotate-token.
- **Tests**: Entra/Okta-shaped PATCH deprovision revokes sessions; cross-org mutation rejected; group→platform-role rejected; `userName eq` dedup; discovery endpoints served.

---

## Phase D — Delegation chain (RFC 8693)

### D1 — token-exchange grant

- **API**: `grant_type=…token-exchange` on `/oauth/token`. Requires `actor_token` (delegation-only profile, P3). Nested `act` claim (reserved, guarded against forgery). Effective scope = `requested ∩ subject_token.scope ∩ client.max_scopes` fail-closed, monotonic non-widening (H1). Hard `act`-depth limit. Mandatory single `resource` (RFC 8707, P2) — reject 0 or >1 on the delegated path only. Tokens: RFC 9068 `typ: at+jwt`, `client_id`, `sub`=client for machine tokens (G7). Short child TTL; sensitive scopes require introspection (revocation).
- **Tests**: delegation narrows scope; re-delegation cannot widen; depth limit enforced; missing/multiple resource rejected; revoked subject token → child introspection fails.

### D2 — RFC 9728 protected-resource metadata

- **API**: `/.well-known/oauth-protected-resource` per protected API; advertises required `aud`. MCP-usable.
- **Tests**: metadata served; aud contract documented.

---

## Subagent orchestration (build order)

Dependency order is strict: **A → B → C → D**, and within a phase the schema/struct lands and compiles before the parallel per-provider CRUD agents run (they edit different files, no conflict). Pattern per schema-bearing step:

1. **Foundation agent** (sequential): define the struct, `Collections` entry, service interface, GraphQL/proto — commit, build-gate.
2. **6 provider agents** (parallel, one file each): implement CRUD for their provider against the struct — each build-gates its own package.
3. **Integration agent** (sequential): wire handlers/resolvers, run `make generate-graphql`, full build + `TEST_DBS=sqlite` test gate, commit.

Every phase ends with the phase's gate (above) green and a commit. Nothing merges to main; the branch is handed to manual testing per `MANUAL_TEST_PLAN.md`.
