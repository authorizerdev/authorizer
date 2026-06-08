# Design: Agentic Delegation Chain (Token Exchange · Attenuation · Audit)

**Scope:** Capabilities #13 (RFC 8693 token exchange), #14 (attenuation / least-privilege), #21 (audit delegation chain) from the agentic-auth capability matrix. This is the **highest-leverage layer after ReBAC** — it answers *"who is asking, with whose borrowed authority, and constrained to what?"*

**Current state (verified in code):** greenfield. No token-exchange grant, no `act` claim, audit has a single actor. `authorization.Principal.MaxScopes` exists as a "delegation ceiling" to build on.

---

## 1. The problem

An agent acts **as itself** **on behalf of a user**, possibly delegated through one or more apps. Three things must be true on every downstream call:

1. **Both identities travel together** — the resource server and audit must see *agent X acting for user Y*.
2. **The agent gets least privilege** — a token downscoped to the task, never the user's full power.
3. **The full chain is recorded** — `app → agent → user` is queryable in audit.

RFC 8693 (OAuth 2.0 Token Exchange) is the standard mechanism. The `act` claim carries the chain.

---

## 2. The `act` claim — the heart of the design

A task-scoped token minted for the agent:

```jsonc
{
  "sub": "user:alice",                  // whose authority is exercised
  "aud": "https://calendar.example",    // bound target (RFC 8707)
  "scope": "calendar:read",             // attenuated, not alice's full scope
  "exp": "<now+5m>",                    // short-lived
  "act": {                              // who is acting
    "sub": "agent:booking-bot",
    "act": {                            // nested: who delegated to the agent
      "sub": "app:concierge"
    }
  }
}
```

- `sub` stays the **user** (authority source); `act.sub` is the **immediate actor**; nested `act` encodes multi-hop delegation.
- **`act` becomes a reserved claim** (alongside `roles`, `scope`, …) in `internal/token/auth_token.go` so `CustomAccessTokenScript` cannot forge it.

---

## 3. Token exchange endpoint (#13)

Extend `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`.

| Param | Meaning |
|---|---|
| `subject_token` (+ `_type`) | the user's token (authority being exercised) |
| `actor_token` (+ `_type`) | the agent's token (the actor) — present ⇒ **delegation** |
| `requested_token_type` | usually `urn:ietf:params:oauth:token-type:access_token` |
| `resource` / `audience` | RFC 8707 target binding (**required** for MCP) |
| `scope` | requested (down)scope |

**Two semantics:**
- **Delegation** (`actor_token` present): mint composite token with `sub=user`, `act` chain. Default agent path.
- **Impersonation** (no `actor_token`): actor fully becomes the subject. **Admin-gated, always audited.** Used sparingly (support tooling), not the agent path.

**Flow:**
```
1. validate subject_token + actor_token (sig, exp, aud, not revoked)
2. resolve actor's delegation ceiling (agent service-account MaxScopes)
3. attenuated_scope = intersection(subject_effective, requested_scope, actor_ceiling)   ← §4
4. mint access_token: sub=subject, act=chain, aud=resource, scope=attenuated, short exp
5. audit: actor=agent, on_behalf_of=user, chain                                          ← §5
```

---

## 4. Attenuation — least privilege per task (#14)

The exchanged token's authority is the **intersection** of three ceilings:

```
effective = subject_permissions  ∩  requested_scope  ∩  agent_delegation_ceiling
```

- **`subject_permissions`** — what the user actually has (scopes + FGA-derived).
- **`requested_scope`** — what the agent asked for (RAR can carry structured detail).
- **`agent_delegation_ceiling`** — `Principal.MaxScopes` on the agent's service account (the existing primitive). An agent can never exceed its registered ceiling even if the user is an admin.

Further constraints stamped on the token:
- **Audience binding** (`aud` = `resource`, RFC 8707) — token works only at the named MCP server / API.
- **Short TTL** — 5 min default for agent tokens (matches roadmap 4.2).
- **(Later)** sender-constraint via DPoP (Phase 5.2).

**FGA interplay:** object-level checks still run at the resource server as `Check(user:alice, relation, object, context={actor: agent, purpose})`. OpenFGA **Conditions** can additionally require the actor be a delegated agent / within a context window. Token attenuation is the *coarse* least-privilege gate; FGA is the *fine* one. Both apply.

---

## 5. Audit delegation chain (#21)

**Schema change (storage tax across all DBs).** Extend `schemas.AuditLog` and `audit.Event`:

```go
// audit.Event — add:
OnBehalfOfID    string  // the subject (user) the actor acted for
OnBehalfOfType  string  // "user" | "agent"
DelegationChain string  // serialized act chain, e.g. "app:concierge>agent:booking-bot>user:alice"
```

- Every action performed with an exchanged token logs **actor + on-behalf-of + chain**.
- Makes "what did `booking-bot` do for Alice last week?" and "every action delegated through `app:concierge`" queryable.
- Schema must be added to all 6 DB implementations + AutoMigrate / collection setup (same multi-DB pattern as any new field).

---

## 6. End-to-end runtime flow

```
User ──OIDC──► Authorizer ──► user token (sub=alice)
App/agent ──token-exchange(subject=user, actor=agent, scope=calendar:read, resource=calendar)──► Authorizer
Authorizer ──► attenuated token (sub=alice, act=[app>agent], aud=calendar, scope=calendar:read, exp=5m)
Agent ──► MCP tool / Calendar API  (presents attenuated token)
Resource server ──► validate aud+scope ──► FGA Check(alice, read, calendar:..., ctx{actor:agent})
Authorizer audit ──► {actor: agent, on_behalf_of: alice, chain: app>agent>alice, action, resource}
User ──► dashboard: view / revoke active agent delegations   (roadmap 4.3)
```

---

## 7. Decisions (LOCKED — principal-engineer calls)

| # | Decision | Locked choice |
|---|---|---|
| DC1 | Delegation vs impersonation | **Both**; impersonation admin-gated + always audited |
| DC2 | Where the agent ceiling lives | Agent **service-account `MaxScopes`** + optional FGA tuples |
| DC3 | `act` claim format | **RFC 8693 nested `act`** (multi-hop chains) |
| DC4 | Token binding | **RFC 8707 `resource`/`aud` required** for exchanged/MCP tokens; DPoP later (Phase 5.2) |
| DC5 | Revocation | **Short TTL (5m) baseline + revocation-list-via-introspection for sensitive scopes** (see §7.1) |

### 7.1 Revocation — the real design (not a footnote)
A two-tier model, reusing the **existing `/oauth/introspect` endpoint**:
- **Default (non-sensitive scopes):** rely on the **5-minute TTL**. Revoking a delegation stops *refresh*; in-flight access tokens expire within the window. Acceptable for the common case.
- **Sensitive scopes (operator-flagged, e.g. `payments:*`):** the exchanged token is marked `sensitive`, and **resource servers MUST call `/oauth/introspect`** before honoring it. Introspection checks a **revocation list** updated the instant a user revokes the delegation (dashboard / `revoke` mutation) → immediate effect, no TTL wait.
- **Distributed propagation:** the revocation list lives in `memory_store` (Redis/DB) so all nodes and the introspect endpoint see revocations consistently. Document the (sub-second) propagation window.

This keeps the hot path cheap (short TTL, no introspection) while giving immediate revocation where it actually matters.

**Prereq dependency (review fix #10):** this design sits on **agent identity + M2M/client-credentials** (roadmap Phase 2 / 4.2) and the **OpenFGA decision core** (FGA_OPENFGA_MIGRATION_PLAN.md). Do not start Wave 2 before those land.

## 8. Phasing (verify-gated)

1. **`act` claim + reserved-claim guard** → verify: minted token carries `act`; custom script cannot overwrite it.
2. **Token-exchange grant (delegation only)** → verify: subject+actor tokens produce composite token with correct `act` + intersected scope + bound `aud`.
3. **Attenuation via agent `MaxScopes`** → verify: agent cannot exceed its ceiling even with an admin subject token.
4. **Audit chain fields (all DBs)** → verify: exchanged-token action logs actor + on_behalf_of + chain; `make test-all-db` green.
5. **Impersonation (admin-gated) + dashboard revocation** → verify: impersonation requires admin + audits; user can revoke an active delegation.

**Dependency:** sits on top of the OpenFGA decision core (FGA_OPENFGA_MIGRATION_PLAN.md) and the M2M/service-account + agent-identity work (roadmap 2.2 / 4.2).
