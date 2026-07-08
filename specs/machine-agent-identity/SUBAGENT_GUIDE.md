# Subagent Guide — PR pipeline for machine-agent-identity

How every PR in this program is planned, built, tested, reviewed, and shipped. Read this + `IMPLEMENTATION_PLAN.md` + `MANUAL_TEST_PLAN.md` before starting any PR.

## The non-negotiable dependency reality

These PRs form a **stack**, not 8 independent parallel merges — each builds on the prior's code:

```
main
 └─ PR1  Client registry: schema + storage (6 providers) + service layer
     └─ PR2  Client registry API: GraphQL _client_* + gRPC admin + kind-aware projection
         └─ PR3  Resolver + client_credentials + discovery metadata + BC1/BC2/BC3 + refresh rotation
             └─ PR4  Secretless auth: TrustedIssuer client_assertion + SPIFFE + K8s  (Phase B)
                 └─ PR5  Organizations + memberships + (org,client) FGA grant + HIGHER_CONSISTENCY  (Phase C1/C2)
                     ├─ PR6  SSO: SAML SP + OIDC broker  (Phase C3)   ← PR6 and PR7 are independent of each other
                     └─ PR7  Inbound SCIM 2.0 server     (Phase C4)   ← both depend on PR5, not on each other
                         └─ PR8  Delegation: token exchange (RFC 8693)  (Phase D)
```

**Where parallelism is safe:**
- **Within a PR**: the 6 storage providers (sql, mongodb, arangodb, cassandradb, dynamodb, couchbase) are separate files — fan out one agent per provider *after* the shared struct/interface lands.
- **PR6 (SSO) and PR7 (SCIM)** both depend only on PR5 (orgs) and touch different code — they can be authored in parallel worktrees off PR5's branch.
- Everything else is sequential: a PR's branch is based on its parent PR's branch (stacked), and each targets `main` (or, if the parent isn't merged yet, retarget once it is). Do NOT base a PR on `main` if it needs a parent PR's code.

## Base branches

- PR1 branches off `origin/main`.
- PRn (n>1) branches off PR(n-1)'s branch. When PR(n-1) merges to main, rebase PRn onto main and retarget the PR base to `main`.
- The current `feat/machine-agent-identity` branch already contains PR1+PR2+part of PR3 material (the merged phase-1 work + the Client rename). PR1/PR2 are carved out of it.

## SDLC every PR follows (owned by a `principal-engineer` lead agent)

1. **Plan** — write a 10-line plan comment on the PR scope: what changes (DB/API/UI), which files, which edge cases from `MANUAL_TEST_PLAN.md` it must cover, what it explicitly defers. Post it in the PR description.
2. **Implement** — on the PR branch, honoring every ground rule below. Schema-bearing PRs: land the struct + `Collections` entry + `Provider` interface first (sequential), THEN fan out per-provider CRUD agents, THEN an integration agent wires handlers/resolvers + runs codegen.
3. **Test** — unit + integration; add regression tests for this PR's edge cases (name them after the finding, e.g. `TestClientAssertion_SSORowRejected` for CR1). `TEST_DBS=sqlite go test -p 1 ./internal/...` green locally; the PR's CI runs the rest.
4. **Review** — dispatch a `security-engineer` pass AND a `principal-engineer` (second-pair) pass on the diff. Every CRITICAL/HIGH/BLOCKER is fixed before requesting human review; SHOULD-FIX either fixed or explicitly deferred with a reason in the PR.
5. **Iterate** — loop 3–4 until build + `make lint` + tests are green and reviews are clean. Then open/update the PR.

## Ground rules (hard gates — a PR is not done until all hold)

- **Never commit to `main`. Never merge. Never force-push a shared branch.** Open the PR; a human merges.
- **Never** weaken the Rev.5 security fixes. The design doc §5.2 findings (BC1–BC3, CR1–CR3, H1–H7, G-gaps) are requirements, not suggestions. Each PR's tests must prove its relevant findings are closed (see `MANUAL_TEST_PLAN.md`).
- Schema change ⇒ all 6 providers + `Collections` + per-provider migration/index. Add a round-trip test asserting every field (incl. any `json:"-"` secret) persists and never serializes out.
- `make generate-graphql` after `schema.graphqls`; `make proto-gen` after `proto/`; commit generated output; PR must have zero codegen drift.
- Transports thin; logic in `internal/service/` behind a `Provider` interface. Admin GraphQL ops `_`-prefixed. Constants in `internal/constants` (no literals in dispatch). gin-context error pattern (#638/#639); fire-and-forget goroutines use `context.WithoutCancel`. Audit events dotted `<actor>.<action>`, success + failure.
- Single-use caches (`jti`, SAML AssertionID, OAuth code) and the authoritative client cache live in the shared `memory_store` (Redis), invalidated cross-instance.
- **Build gate**: `go build ./... && make lint-go && go vet ./...`. **Test gate**: `TEST_DBS=sqlite go test -p 1 ./internal/...`. **Full gate before marking ready**: `make test-all-db` green (or CI equivalent).
- Commit messages: conventional, no Co-Authored-By, no AI attribution. PR descriptions: no AI-branding.

## Per-PR scope (summary — full detail in IMPLEMENTATION_PLAN.md)

| PR | Scope | Key edge-case tests |
|----|-------|---------------------|
| PR1 | `Client` schema + `Collections` + 6-provider CRUD + service layer | round-trip persists secret + never leaks; unique client_id; kind default |
| PR2 | `_client_*` GraphQL + gRPC admin, kind-aware projection, one-time secret reveal, admin-suppliable client_id | secret never in get/list; kind immutable; dup client_id rejected |
| PR3 | resolver (secret/basic/none), `client_credentials`, discovery metadata, BC1/BC2/BC3, refresh rotation | existing token/session unchanged (BC1/BC2); dual-auth rejected; scope-subset |
| PR4 | TrustedIssuer client_assertion, SPIFFE, K8s | CR1 SSO-row rejected; replay; subject exact-match; alg:none; issuer collision |
| PR5 | Organizations, memberships, (org,client) FGA grant, HIGHER_CONSISTENCY | C3 org from grant not request; revoked grant stops stamping; membership unique |
| PR6 | SAML SP + OIDC broker | XSW; unsigned; cross-org audience; replay; mix-up (iss); no email-only link |
| PR7 | Inbound SCIM 2.0 | CR2 platform-role rejected; H6 cross-org mutation rejected; deprovision revokes sessions |
| PR8 | token exchange (RFC 8693) | attenuation non-widening; depth limit; single resource; revocation via introspection |

## Orchestration order (what the coordinator dispatches)

1. Gate: `make test-all-db` green on `feat/machine-agent-identity` (the current base). ← do this FIRST.
2. Carve PR1 and PR2 from the current branch, open them (stacked), run their review passes.
3. PR3 on top; review.
4. PR4 on top; review (security-heavy).
5. PR5 on top; review.
6. PR6 + PR7 in parallel worktrees off PR5; review each.
7. PR8 on top of PR5 (after PR6/PR7 or in parallel — it doesn't touch SSO/SCIM); review.

The coordinator verifies each PR's gates itself before opening it — never trust a subagent's self-report on a security-critical PR without re-running build + the relevant tests.
