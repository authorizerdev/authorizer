# Authorizer Migration Tool — Design Spec

**Status:** Draft for review
**Date:** 2026-06-08
**Author:** Principal Engineering
**Scope:** A zero-downtime migration tool to move users, credentials, and configuration into Authorizer from Auth0, Clerk, WorkOS, Keycloak, Okta, and SuperTokens — plus the Authorizer core APIs required to support it.

---

## 1. Problem & goals

Authorizer (as a SaaS / self-hosted offering) wants to win customers off incumbent auth providers. The single biggest adoption blocker is migration risk: moving an existing user base without **(a)** forcing password resets, **(b)** a maintenance window, or **(c)** losing roles/MFA/social links.

### Goals
1. Migrate **users + credentials + social links + roles + MFA + org/tenant + client config** from six sources.
2. **Zero downtime, zero forced reset** for the common case (coexistence migration with live cutover and rollback-before-cutover).
3. Data never transits Authorizer SaaS unnecessarily — extraction runs in the customer's network.
4. Idempotent, resumable, inspectable, dry-runnable.

### Non-goals (v1)
- Auto-applying org/tenant and OAuth-client/connection config into Authorizer (v1 = **report-only**, see §9).
- True "no re-login" session continuity via foreign-JWKS bridging (v1 = **one silent re-auth at cutover**, see §7).
- Reversible-after-cutover write-journaling (v1 = **forward-only after a validated bake**, see §7).
- WebAuthn/passkey migration (cryptographically bound to original origin — re-enroll required).

---

## 2. Research findings: what each source actually exposes

Verified against current provider docs / source (June 2026). This table drives every connector's behavior.

| Source | User profiles | Password hashes (self-serve?) | Hash algo | MFA/TOTP seeds | Social links | Roles/RBAC | Orgs/Tenants |
|---|---|---|---|---|---|---|---|
| **Keycloak** | ✅ realm export / Admin API | ✅ **Yes** — realm-export JSON | PBKDF2-sha256/512 (+iter+salt, 64-byte dk) | ✅ **Yes** — seed in export | ✅ `federatedIdentities` | ✅ realm/client roles, groups | realm = tenant |
| **SuperTokens** | ✅ Core API (`GET /users`) | ✅ **Yes** — direct DB read (self-hosted) | **bcrypt default**, or argon2id | ✅ **Yes** — DB `totp_user_devices.secret_key` | ✅ `thirdParty` per loginMethod | ✅ roles recipe | ✅ multitenancy API |
| **Clerk** | ✅ Backend API / Dashboard CSV | ✅ **Yes** — Dashboard CSV incl. hashes | bcrypt | ❌ no | ✅ `external_accounts` | ✅ org roles/perms | ✅ organizations |
| **Auth0** | ✅ `POST /jobs/users-exports` | ⚠️ **support ticket only** (PGP, paid tier) | bcrypt (`$2a/$2b$`) | ⚠️ same PGP ticket | ✅ `identities[]` | ✅ Authorization Core | ✅ Organizations |
| **Okta** | ✅ `GET /api/v1/users` | ❌ **Never** | n/a | ❌ no | ✅ IdP links | ✅ groups, roles | single org |
| **WorkOS** | ✅ User Mgmt API | ❌ **Never** (reset/lazy) | n/a | ❌ no | ✅ AuthKit identities | ✅ roles | ✅ orgs + SSO conns |

**Conclusion:** Hashes are obtainable from three sources directly (Keycloak/SuperTokens/Clerk), one with heavy effort (Auth0), never from two (Okta/WorkOS). Therefore the tool **must** support both a bulk-hash path and a live lazy/JIT path. Confirmed cross-reference: WorkOS's own "migrate from X" docs document the same realities and fall back to password-reset; we improve on that with live lazy verification.

### Key source endpoints (for connectors)
- **Auth0:** `POST /api/v2/jobs/users-exports` (NDJSON, 24h TTL, 60s download link); `GET /roles`, `/roles/{id}/permissions`, `/users/{id}/roles`; `/organizations/*`; `/clients` (+`read:client_keys` for secrets); `/connections` (+`read:connections_options` for social/SAML keys); `/resource-servers`; `/users/{id}/enrollments` (MFA metadata only). Mgmt token via M2M client-credentials, 24h expiry. **Password hashes + MFA secrets:** PGP support-ticket export only (`export-password-hashes-and-mfa-secrets`), not Free tier.
- **Clerk:** `GET /v1/users` (Backend API, bcrypt `password_digest`/`password_hasher`); Dashboard CSV export includes hashes (since 2024-10-23); `/v1/organizations` + memberships; trickle-migration documented. Insecure hashers transparently upgraded to bcrypt on first login.
- **WorkOS:** `GET /user_management/users` (cursor pagination); `/organizations`, `/organization_memberships`; `/connections` (SSO SAML/OIDC); `/directories` (SCIM). No hash export → lazy/reset. Import side accepts bcrypt/scrypt/firebase-scrypt/ssha/pbkdf2/argon2 in PHC.
- **Keycloak:** realm export via `kc.sh export --users different_files` (credentials included) — `credentials[].secretData{value,salt}` + `credentialData{hashIterations,algorithm}` (PBKDF2, **64-byte derived key**, read per-credential iteration count); OTP creds carry the TOTP seed; `federatedIdentities`; `realmRoles`/`clientRoles`; groups; `/clients` (+secrets), `/identity-provider/instances` (+secrets). Admin token via `admin-cli` client-credentials.
- **Okta:** `GET /api/v1/users` (limit 200, link-header cursor); `/idps` + `/idps/{id}/users`; `/users/{id}/factors` (no secrets); `/groups`; `/apps` (+`/apps/{id}/users`). SSWS or OAuth2 scopes (`okta.users.read`, …). **No hash export** → Okta Password Import Inline Hook is the lazy pattern (we mirror it in reverse).
- **SuperTokens:** Core API `GET /users` (`paginationToken` loop), `/recipe/roles`, `/recipe/multitenancy/tenant/list`, `/recipe/user/metadata`; **hashes + TOTP secrets via direct DB read** (`emailpassword_users.password_hash`, `totp_user_devices.secret_key`); bulk-import API shape (`loginMethods[].passwordHash`+`hashingAlgorithm` ∈ {argon2,bcrypt,firebase_scrypt}, `totpDevices[].secretKey`, `userRoles`, `userMetadata`, `externalUserId`) is the reference for our own import API.

---

## 3. Architecture

Three components.

### (a) `authorizer-migrate` CLI (Go single binary)
Matches Authorizer's stack. Runs in the **customer's network** so source data never transits Authorizer SaaS. Pipeline:

```
  extract                 transform/normalize          load
┌──────────┐   source     ┌────────────────────┐      ┌──────────────────┐
│ connector│──API/DB────▶ │  <source>.amf.json │ ───▶ │ Authorizer       │
│ (source  │              │  (canonical AMF,    │      │ _import_users    │
│  creds)  │              │   inspectable)     │      │ (admin creds)    │
└──────────┘              └────────────────────┘      └──────────────────┘
```

Commands:
- `authorizer-migrate extract --from <provider> --config <creds.yaml> --out users.amf.json [--since <ts>]`
- `authorizer-migrate validate users.amf.json` (schema + referential integrity, offline)
- `authorizer-migrate load --in users.amf.json --authorizer <url> --admin-secret … [--dry-run] [--resume]`
- `authorizer-migrate report --in users.amf.json` (org/RBAC/client-config report, §9)

Properties: **idempotent upsert** (`source`+`externalUserId`), **resumable** (checkpoint file), **dry-run**, **delta** (`--since`).

### (b) AMF — Authorizer Migration Format
One canonical JSON schema all connectors normalize into. Decouples extract (source creds) from load (target creds), gives a human review/edit/diff checkpoint, makes the pipeline idempotent + resumable. Sketch:

```jsonc
{
  "amfVersion": "1.0",
  "source": { "provider": "keycloak", "exportedAt": 1749312000, "realmOrTenant": "acme" },
  "users": [
    {
      "externalUserId": "f8a...",            // stable source id → idempotency key
      "email": "jane@acme.com",
      "emailVerified": true,
      "phoneNumber": null,
      "givenName": "Jane", "familyName": "Doe",
      "picture": "https://...",
      "roles": ["admin", "billing"],
      "credential": {
        "type": "password_hash",             // password_hash | none (lazy) | reset
        "algorithm": "pbkdf2-sha256",        // bcrypt | pbkdf2-sha256 | pbkdf2-sha512 | argon2id | scrypt | firebase-scrypt
        "hash": "base64...",                  // normalized to PHC string where possible
        "params": { "iterations": 27500, "salt": "base64...", "keyLen": 64 }
      },
      "identities": [
        { "provider": "google", "providerUserId": "104...", "email": "jane@gmail.com" }
      ],
      "mfa": [
        { "type": "totp", "secret": "BASE32SEED", "algorithm": "SHA1", "digits": 6, "period": 30 }
      ],
      "appData": { "sourceOrg": "acme", "legacyId": "..." },
      "createdAt": 1700000000, "updatedAt": 1749000000
    }
  ],
  "lazyMigration": {                          // present when source can't export hashes
    "enabled": true,
    "originProvider": "okta",
    "verifyVia": "password-grant"             // how Authorizer re-verifies on first login
  }
}
```

`config-report` (orgs, roles tree, OAuth clients, connections) is emitted to a **separate** `*.report.json` consumed by `report`, not by `load` (v1 report-only).

### (c) Authorizer core additions
The "missing APIs." See §4–§6.

---

## 4. New Authorizer APIs (the missing surface)

Authorizer today has **no bulk import** and **no create-user-with-prehashed-password** path (`signup` hashes plaintext; `invite_members` invites). We add:

### 4.1 `_import_users` — admin GraphQL mutation (async job)
- Admin-authenticated (admin secret), behind the existing `_`-prefixed admin namespace.
- Accepts a batch (≤ N per call, e.g. 1000) of canonical AMF user records.
- Creates users with **pre-hashed passwords + algorithm metadata**, identities, roles, TOTP devices, `app_data`, and `externalUserId`+`source` for idempotency.
- **Idempotent upsert:** re-running with the same `source`+`externalUserId` updates instead of duplicating (enables delta sync).
- Returns a `jobID`.

```graphql
mutation { _import_users(params: ImportUsersInput!): ImportJob! }

input ImportUsersInput {
  source: String!                 # "auth0" | "keycloak" | ...
  users: [ImportUserInput!]!
  upsert: Boolean = true
}
input ImportUserInput {
  external_user_id: String!
  email: String
  email_verified: Boolean
  phone_number: String
  given_name: String
  family_name: String
  picture: String
  roles: [String!]
  password_hash: String          # optional; omit for lazy/reset users
  password_hash_algorithm: String # bcrypt | pbkdf2-sha256 | pbkdf2-sha512 | argon2id | scrypt | firebase-scrypt
  password_hash_params: String   # JSON: iterations, salt, keyLen, memory, parallelism, etc.
  identities: [ImportIdentityInput!]
  mfa: [ImportMFAInput!]
  app_data: String               # JSON
  created_at: Int64
  updated_at: Int64
}
```

### 4.2 `_import_status(jobID)` + job resource
Large tenants import async. Poll job progress and inspect per-row failures (mirrors SuperTokens' `bulk-import/users?status=FAILED`).

```graphql
query { _import_status(job_id: ID!): ImportJob! }
type ImportJob { id: ID!  status: ImportStatus!  total: Int!  succeeded: Int!  failed: Int!  errors: [ImportRowError!]! }
enum ImportStatus { QUEUED PROCESSING COMPLETED COMPLETED_WITH_ERRORS FAILED }
type ImportRowError { external_user_id: String!  message: String! }
```

---

## 5. User schema change + multi-algorithm verifier

**Today:** `User.Password *string` (bcrypt, `bcrypt.DefaultCost`), no algorithm column → a Keycloak PBKDF2 or SuperTokens argon2id hash cannot be verified after import.

**Change (decision: add algo-metadata + multi-algo verify with transparent upgrade):**

1. Add columns to `User` (all providers): `password_hash_algorithm *string`, `password_hash_params *string` (JSON). Nullable; null/empty ⇒ legacy bcrypt (back-compat — existing rows untouched).
2. Add a `crypto` verifier registry: `bcrypt` (native, existing), `pbkdf2-sha256`, `pbkdf2-sha512`, `argon2id`, `scrypt`, `firebase-scrypt`. Each verifies a candidate password against `(hash, params)`.
3. **Login path change:** on password verify, dispatch by `password_hash_algorithm`. On the **first successful** login against a non-bcrypt hash, **transparently re-hash to bcrypt** (`bcrypt.DefaultCost`), persist, and clear the algorithm/params columns. This is Clerk's upgrade model — within a tail period the whole base converges to native bcrypt and the foreign-algo code is exercised only transiently.

**Verification:** import a hand-written AMF with one bcrypt, one pbkdf2-sha256, and one argon2id user; log in as each with the original password and get a session with **no reset**; confirm the non-bcrypt rows flip to bcrypt + null metadata after first login.

**Security notes:** foreign-algo verifiers are constant-time-compared; params are validated (bounded iteration/memory to prevent a malicious AMF from triggering resource exhaustion); the existing dummy-bcrypt timing-equalization on login (`internal/graphql/login.go`) is extended to cover the dispatch so non-existent-user timing doesn't leak.

---

## 6. Lazy / JIT migration-source (the zero-reset coexistence engine)

For Okta & WorkOS (no hash export) **and** for any user whose password changed after the last delta, Authorizer verifies the password **live against the origin provider** on first login.

**Decision: lives in Authorizer core** as a configurable *migration source* (not an external webhook), mirroring Auth0/Okta's own inline-hook pattern but in reverse.

- New config object (DB-config, like other Authorizer settings): `migration_source { provider, base_url, credentials, verify_method }` where `verify_method` ∈ `{ password-grant, ropc, verify-endpoint }`.
- A user imported via the lazy path carries `credential.type = none` and a dedicated nullable `User.migration_source *string` column (checked on the hot login path — avoids parsing `app_data`). Non-null ⇒ this user still needs origin verification.
- **Failed-login hook:** when local verification fails (or there is no local hash) for a user with non-null `migration_source`, Authorizer calls the origin's password-grant/verify API with the submitted credentials. On success → mint an Authorizer session, **store the password as bcrypt**, null out `migration_source`. On failure → normal invalid-credentials.
- Promoted from "Okta/WorkOS only" to a **general coexistence primitive** usable by all six connectors.

**Verification:** configure an Okta migration source; import an Okta user with `credential.type=none`; log into Authorizer with the user's real Okta password → success, session issued, row now has a bcrypt hash and null `migration_source`; second login verifies locally (no origin call).

---

## 7. Zero-downtime strategy

Principle: **never a moment where neither provider can authenticate; never a forced reset or maintenance window.** Four layers:

1. **Bulk pre-seed** — initial export → AMF → `_import_users`. Authorizer runs as a **shadow**; app still points at the old provider. Most users immediately authenticatable where hashes are exportable.
2. **Repeatable delta sync** — `extract --since <ts>` + idempotent `load` upsert, scheduled (e.g. hourly), keeps the shadow current as users sign up / edit profiles in the old system.
3. **Live lazy verify-against-origin (§6)** — the correctness guarantee: whatever the user's *current* origin password is, it validates live on first login. **Delta sync therefore never needs to carry passwords for correctness** — only profile freshness and lazy-load reduction.
4. **Staged cutover + rollback:**

```
Coexist:      app → OLD (authoritative).  CLI bulk + delta → Authorizer (shadow, READ-ONLY from source).
Cutover:      flip app auth → Authorizer (config/DNS). Authorizer authoritative for writes.
              Lazy hook verifies stragglers live against OLD (read-only). No more writes to OLD.
Decommission: after tail period + final reconciliation, disable OLD + lazy hook.
```

Because coexistence is **read-only from the source**, rollback *before* cutover is a no-op and the source is never mutated.

### Decided tradeoffs
- **Session continuity = one silent re-auth at cutover.** Old-provider JWTs aren't trusted post-cutover; users re-authenticate once (seamless via bulk hash or lazy verify — no reset, no error). No foreign-JWKS bridging in v1.
- **Rollback = forward-only after a validated bake.** Cutover begins with a canary (e.g. 1–2% traffic or a pilot tenant) that is validated; once full cutover is confirmed healthy, forward-only. No write-journaling / reverse-sync in v1.

### Deliverable: cutover runbook
A checklist doc (pre-flight credential/scope checks, pre-seed, delta cadence, canary, validation queries, go/no-go, rollback-before-cutover procedure, decommission criteria) shipped alongside the CLI.

---

## 8. Per-provider connector behavior

| Connector | Profiles | Password disposition | MFA | Notes |
|---|---|---|---|---|
| **Keycloak** | realm export | **bulk hash** (PBKDF2 → multi-algo verify) | **bulk TOTP seed** | best case; per-credential iterations; realm→instance |
| **SuperTokens** | Core API + DB | **bulk hash** (bcrypt direct / argon2id via multi-algo) | **bulk TOTP seed** (DB) | DB read for hash+seed; account-linking → identities |
| **Clerk** | Backend API / CSV | **bulk hash** (bcrypt) | re-enroll | CSV includes hashes |
| **Auth0** | export job | **bulk hash if PGP export obtained**, else **lazy** | bulk if PGP, else re-enroll | NDJSON→JSON; identities[] |
| **Okta** | Users API | **lazy** (verify-against-origin) | re-enroll | mirrors Okta Password Import Hook in reverse |
| **WorkOS** | User Mgmt API | **lazy** | re-enroll | SSO/Directory config → report |

Each connector: paginates fully (no silent caps — log dropped/over-limit), normalizes to AMF, supports `--since`.

---

## 9. Scope mapping for orgs / RBAC / client-config (v1 = report-only)

- **Orgs/tenants:** Authorizer is single-tenant per instance. Map **1 Keycloak realm / 1 Auth0 tenant / 1 WorkOS org-set → 1 Authorizer instance.** Multi-org sources emit a **mapping report**; org membership is preserved as a role and/or `app_data.sourceOrg` namespace (no silent loss).
- **RBAC:** source roles/permissions → Authorizer roles + the new **FGA permission model**; emitted to the report with a proposed mapping the customer confirms.
- **App/client & connection config** (OAuth clients, social keys, SAML/OIDC enterprise connections): exported into the report (secrets flagged, never auto-written). Customer applies via Authorizer's DB-config. **Rationale:** these are low-volume, high-blast-radius config objects; auto-writing them into a single-tenant instance is risky and provider-shape-specific. Auto-apply is a post-v1 candidate.

---

## 10. Phasing & milestones (each ships + verifies independently)

| Phase | Deliverable | Verify |
|---|---|---|
| **0** | User schema columns + multi-algo verifier + transparent upgrade (§5) | bcrypt+pbkdf2+argon2id users log in from hand-written AMF, no reset; non-bcrypt flips to bcrypt post-login |
| **1** | `_import_users` + `_import_status` async job (§4) | import 10k-row AMF; idempotent re-run upserts; failures surfaced per-row |
| **2** | AMF spec + CLI skeleton (`extract`/`validate`/`load`, dry-run, resume, `--since`) | round-trip a sample AMF; resume after kill; dry-run mutates nothing |
| **3** | Connectors (hash-friendly order): Keycloak → SuperTokens → Clerk → Auth0 | each extracts to valid AMF; bulk-hash users log in post-load with no reset |
| **4** | Migration-source + failed-login hook (§6); Okta + WorkOS connectors | Okta user logs into Authorizer with old password → silently upgraded |
| **5** | Zero-downtime cutover runbook + delta-sync scheduling guide (§7) | dry-run a full coexist→delta→cutover on a pilot tenant |
| **6** | `report` command: orgs/RBAC/client-config (§9) | report lists every org/role/client/connection with proposed mapping |

---

## 11. Risks & open questions

- **Auth0 hash export is gated** (PGP + paid tier + CISO sign-off). For customers who can't/won't, Auth0 falls to the lazy path — document this prominently; it's the main "it depends" in the matrix.
- **argon2id params** must be carried exactly (memory/iterations/parallelism); a mismatch silently fails verification. Validate on `load`.
- **Keycloak iteration counts are per-credential** and version-dependent — never hardcode; read each credential. Confirm 64-byte derived-key length against the customer's Keycloak version.
- **Delta-sync window for profile writes** (not passwords) made in the old system between last delta and cutover are lost unless cutover stops old-system writes — handled by the "old is authoritative until cutover, then writes move to Authorizer" rule; flag in runbook.
- **Rate limits** (Auth0 Mgmt API, Okta org-wide concurrency) — connectors must backoff; large exports use the async export job where available, not page-scan.
- **PII handling** — AMF files contain hashes/seeds/PII; CLI writes them `0600`, supports encryption-at-rest for the AMF file, and the runbook mandates secure deletion post-migration.

---

## 12. Out of scope (v1)
WebAuthn/passkey migration; foreign-JWKS session bridging; reversible-after-cutover write-journaling; auto-applying org/client config; non-listed source providers (Cognito/Firebase/Supabase — future connectors, AMF already accommodates them).
