# Machine & Agent Identity — Manual Test Plan

Run against branch `feat/machine-agent-identity`. Each phase has a **regression** block (existing behavior must be unchanged) and a **new-feature** block with **edge cases**. `[ ]` = check. Expected results are stated; anything else is a bug.

Setup: `make dev` (SQLite, embedded dev keys) unless a phase says otherwise. Note the deployment's `Config.ClientID`/`ClientSecret` — several checks compare against them.

---

## Phase A — Client registry

### A. Regression (must be UNCHANGED)
- [ ] Existing login (email+password, authorization_code+PKCE) completes; access token issued.
- [ ] Decode the access token: `aud` == the deployment's `Config.ClientID` (BC1). **Edge:** a token issued *before* the upgrade still validates after it (no forced re-login).
- [ ] Existing browser session cookie still decrypts after upgrade (BC2). **Edge:** rotate the *client's* stored secret via the new admin API → existing sessions still decrypt (cookie key not coupled).
- [ ] `GET /.well-known/openid-configuration` — same issuer/endpoints; `grant_types_supported` now additionally lists `client_credentials`. JWKS shape unchanged; `kid` == `Config.ClientID`.
- [ ] `POST /oauth/introspect` for a live token → `active: true`, `client_id` == `Config.ClientID`, same fields as before.
- [ ] `POST /oauth/revoke` with a valid refresh token → `200 {}`; token no longer works. **Edge:** revoke with a garbage token → still `200 {}` (no oracle).
- [ ] `/graphql` with **no** `X-Authorizer-Client-ID` header still works (back-compat allowance).

### B. New feature: client registry
- [ ] Boot the server twice → exactly one reserved client row exists (idempotent seed). **Edge:** start two instances at once → still one row (no duplicate).
- [ ] Admin creates an `interactive` client via `_create_client`; secret revealed exactly once. **Edge:** fetch/list it again → secret/hash never present in the response.
- [ ] Admin creates a `service_account` client; **Edge:** attempt `_update_client` to change `kind` → rejected (immutable).
- [ ] Create a client with an admin-supplied `client_id` → that exact id is used. **Edge:** supply a `client_id` that already exists → rejected (unique).
- [ ] Create a public `interactive` client (no secret) → PKCE required on its authorize flow; **Edge:** authorization_code without PKCE for it → rejected.

### C. New feature: client_credentials
- [ ] `service_account` authenticates at `/oauth/token` (grant `client_credentials`) with its revealed secret → token issued, no refresh token (RFC 6749 §4.4.3). **Edge:** request a scope outside `max_scopes` → rejected. **Edge:** wrong secret → `invalid_client`, and timing is indistinguishable from an unknown client (constant-time).
- [ ] **Edge:** present BOTH `client_secret_post` and a `client_assertion` in one request → rejected (RFC 6749 §2.3).
- [ ] Failed `client_credentials` writes an audit event `token.client_credentials_failed`; success writes `token.client_credentials`.

---

## Phase B — Secretless client auth (K8s / SPIFFE)

Setup: a `client_assertion_trust` row bound to a test issuer (a local JWKS you control), `expected_subject` pinned, `audience` == token-endpoint URL.

### Regression
- [ ] All Phase A checks still pass.

### New feature + edge cases
- [ ] Present a valid signed assertion (correct iss/aud/sub/exp) as `client_assertion` → authenticated, token issued.
- [ ] **Edge (CR1, the critical):** create an `sso_oidc` row for an org at the *same issuer URL*; present a token from it as a `client_assertion` → **rejected** (lookup is `kind=client_assertion_trust` scoped; the SSO row must never authenticate a client).
- [ ] **Edge (replay, H4):** replay the same assertion twice → second rejected (`jti` single-use, or `(sub,iat,exp)` key when no `jti`).
- [ ] **Edge (H3):** assertion with a subject that only prefix-matches the pinned pattern (e.g. `prod-evil` vs `prod`) → rejected (exact/anchored match).
- [ ] **Edge:** `expected_subject` left empty on the row → the row is deny-all (no assertion authenticates).
- [ ] **Edge (H4 lifetime):** assertion with `iat` far in the past but near-future `exp` → rejected (`exp − iat` exceeds ceiling).
- [ ] **Edge (G8):** assertion signed with `alg:none` → rejected.
- [ ] **Edge (aud):** assertion whose `aud` is a generic issuer string, not the exact token-endpoint URL → rejected.
- [ ] **Edge (H5):** issuer URL `https://kubernetes.default.svc` → rejected at row creation (must be globally unique/external).
- [ ] Only a platform super-admin can create a `client_assertion_trust` row; an org admin cannot.

---

## Phase C — Organizations, SSO, SCIM

### Regression
- [ ] All Phase A/B checks still pass; global (non-org) users unaffected.

### C1 Organizations & membership
- [ ] Create org, add a user as member with a per-org role. **Edge:** add the same user twice → rejected (unique `(org_id, user_id)`). **Edge:** same user is admin in Org A, viewer in Org B → both roles independent.

### C2 (org, client) M2M grant
- [ ] Grant a `service_account` client to Org A → its `client_credentials` token now carries `org_id: A`. **Edge (C3):** request a token with `organization=B` for a client granted only to A → rejected (org_id from the grant, not the request).
- [ ] **Edge (MED-1):** revoke the grant, immediately request a token → `org_id` no longer stamped (requires HIGHER_CONSISTENCY; if it still stamps for a window, that's the bug this gate exists for).
- [ ] **Edge (bidirectional consent):** an Org-A admin tries to grant a client they don't own without the client owner's opt-in → rejected.

### C3 SSO (SAML SP + OIDC)
- [ ] Configure an org SAML connection; a valid signed assertion logs a user in and JIT-provisions. **Edge (XSW):** an assertion where the signature covers a different element than the consumed one → rejected. **Edge:** unsigned assertion → rejected. **Edge:** assertion with Org B's Audience presented at Org A's ACS → rejected. **Edge:** replay the same AssertionID → rejected. **Edge:** SP-initiated response with a mismatched/absent InResponseTo → rejected.
- [ ] OIDC org connection logs in. **Edge (mix-up, G3):** a response bearing a different `iss` than the connection dispatched to → rejected.
- [ ] **Edge (account takeover):** an org IdP asserts an `email` that collides with an existing global user → NOT silently linked (namespaced `(org_id, issuer, sub)`).

### C4 Inbound SCIM
- [ ] Provision a user via SCIM POST with the org's bearer token → user created with `external_id`. **Edge (dedup):** IdP does `GET /Users?filter=userName eq "x"` before create → returns the existing user, no duplicate.
- [ ] Deprovision via `PATCH active:false` → user disabled AND active sessions/refresh tokens revoked **immediately**. **Edge:** the deprovisioned user's still-held access token fails introspection.
- [ ] **Edge (CR2):** a SCIM group that maps to a platform/global role → rejected/ignored (org-namespaced roles only).
- [ ] **Edge (H6):** use Org A's SCIM token to mutate an Org B user (via id in the path/payload) → rejected (org derived from token only).
- [ ] `/ServiceProviderConfig`, `/Schemas`, `/ResourceTypes` served. **Edge:** bad bearer token → `401`, constant-time.

---

## Phase D — Delegation (token exchange)

### Regression
- [ ] All prior checks pass.

### New feature + edge cases
- [ ] `grant_type=token-exchange` with subject_token + actor_token + one `resource` → new token with nested `act`, audience-bound. **Edge:** omit `actor_token` → rejected (delegation-only profile). **Edge:** omit `resource` or pass two → rejected (single-resource profile).
- [ ] **Edge (attenuation, H1):** request a scope broader than the subject_token's scope → narrowed to the intersection, never widened. **Edge:** re-exchange a delegated token requesting the original broad scope → still cannot widen.
- [ ] **Edge (depth):** build an `act` chain past the configured depth limit → rejected.
- [ ] Token carries `typ: at+jwt` and `client_id` (RFC 9068). **Edge (revocation):** revoke the subject token → a child token's introspection for a sensitive scope returns `active:false`.
- [ ] `/.well-known/oauth-protected-resource` served for a protected API with the required `aud`.

---

## Cross-cutting (run after all phases)
- [ ] `make test` (full sqlite suite) green.
- [ ] `make test-all-db` green across all 7 backends.
- [ ] `make lint` clean.
- [ ] Multi-instance: run 2 instances sharing Redis (`TEST_ENABLE_REDIS=1` style) → jti replay rejected across instances; a client disabled on instance 1 is rejected on instance 2 within the cache TTL.
- [ ] No secret/hash ever appears in any GraphQL/REST/gRPC response, any log line, or any webhook payload (grep the audit/webhook logs).
