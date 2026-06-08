# FGA → OpenFGA + Agentic Auth: Agent Fleet & Execution Plan

How a small fleet of specialized agents carries the OpenFGA migration and agentic-auth program forward **to standard, with verification at every gate**. This is the orchestration layer over the design docs.

**Source-of-truth docs (agents MUST read before acting):**
- `FGA_OPENFGA_MIGRATION_PLAN.md` — phased plan + LOCKED decisions (D1–D4, store, why-logs)
- `AGENTIC_DELEGATION_DESIGN.md` — Wave 2 delegation (DC1–DC5)
- `ENTERPRISE_AUTHZ_MODEL.md` — the OpenFGA model + Acme worked example
- `ROADMAP_V2.md` — Agentic Authorization Track (4 waves)

---

## The fleet

| Agent | Model | Owns | Reads | Produces |
|---|---|---|---|---|
| **authz-researcher** | opus | Deep, adversarially-verified research on OpenFGA, Zanzibar, Auth0 parity, and standards (RFC 8693/8707/9728, MCP, CIBA, AuthZEN) | the web + design docs | cited research briefs that gate design choices |
| **fga-engineer** | opus | Wave 1 — engine SPI, embedded/external OpenFGA, model/tuple/check APIs, GraphQL `_fga_*`/`fga_*`, session/validate changes, dashboard wiring, SDK FGA surface | migration plan, openfga-modeling skill | working code + tests, one verified phase at a time |
| **delegation-engineer** | opus | Wave 2 — RFC 8693 token exchange, `act` claim, attenuation, audit delegation chain | delegation design, agentic-auth-standards skill | working code + tests, security-first |

**Existing agents reused (do not duplicate):**
- `principal-engineer` — owns any change touching >1 subsystem; the default driver if a program agent isn't a better fit.
- `security-engineer` — **mandatory second pass** on every FGA/delegation PR (auth-sensitive).
- `doc-writer` — migration guide, API docs, Auth0-import guide.

**Domain skills (auto-load on matching work):**
- `openfga-modeling` — DSL/tuple patterns, enterprise model, the **verified embed API**, check/list_objects, conditions.
- `agentic-auth-standards` — the standards + the LOCKED decisions, so no agent re-litigates them.

---

## Execution waves (each gated by `Verify`)

### Wave 0 — Research & validate *(authz-researcher)*
- Confirm current OpenFGA embed API + SQLite/Postgres datastore bootstrap (the one remaining spike step).
- Track standards deltas (MCP spec, ID-JAG draft status, AuthZEN).
- → **Verify:** every claim feeding a design decision is cited and adversarially checked. Spike code compiles & runs.

### Wave 1 — Decision core *(fga-engineer → security-engineer)*
Follows `FGA_OPENFGA_MIGRATION_PLAN.md` Phases 1→7, **two-release rollout** (both engines behind `--authorization-engine`, remove old in N+1).
- → **Verify per phase:** the phase's own Verify gate + `go build ./...` + `make test-all-db` (proves no DB-impl dangling refs) + security-engineer review.

### Wave 2 — Delegation core *(delegation-engineer → security-engineer)*
Follows `AGENTIC_DELEGATION_DESIGN.md`. **Prereq:** agent identity + M2M (roadmap Phase 2). Do not start before it lands.
- → **Verify:** `act` chain correct, attenuation can't exceed ceiling, revocation works for sensitive scopes, audit chain queryable, security review.

### Waves 3–4 — Async/custody + Enterprise hardening
CIBA+RAR, Token Vault, MCP/ID-JAG, JIT/guardrails. New research brief per capability before build.

---

## Handoff protocol (research → implement → review)

1. **Research first.** No engineer agent starts a capability without a current `authz-researcher` brief (or a cited section in the design docs). "Don't assume" is enforced here.
2. **Implement to the plan.** Engineer agents follow the phased plan + locked decisions verbatim. Deviation requires an explicit, written rationale appended to the relevant design doc — not a silent change.
3. **Verify before claiming done.** Build + tests + the phase's Verify gate. No "should work."
4. **Security review is mandatory**, not optional, for every FGA/token/delegation PR (`security-engineer`).
5. **Docs follow** (`doc-writer`) once an API is frozen.

---

## Definition of Done (program standard — non-negotiable)

A change is done only when ALL hold:
- [ ] Matches the LOCKED decisions (or amends the design doc with rationale).
- [ ] `go build ./...` green; `make generate-graphql` run if schema changed.
- [ ] Tests written and passing; `make test-all-db` green for storage-touching changes.
- [ ] Auth-sensitive code reviewed by `security-engineer`.
- [ ] Principal pinned to token `sub` on every `fga_*` runtime check; admin-gating on `_fga_*` and model edits; fail-closed on engine error.
- [ ] No secrets in logs; no stale-allow cache path (cache only in embedded mode).
- [ ] Verified against the phase's `Verify` gate with real output, not assertion.

---

## Guardrails the agents must respect (lessons already paid for)
- **FGA store is SQL-only** (SQLite single-node / external Postgres for HA). Do not attempt Mongo/Dynamo adapters.
- **Don't double-cache** decisions in external mode.
- **`required_relations` is the new fine path; `roles` filter stays for coarse** — never force capabilities into ReBAC except via the singleton-object pattern.
- **Two-release removal** of the old engine, not big-bang.
- **`go.mod` probe passed** (openfga v1.17.1, sqlite→v1.51.0 compiles clean) — but re-run `make test-sqlite` on the integration branch.
