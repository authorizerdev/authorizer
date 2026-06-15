# CLAUDE.md

Project instructions for AI coding tools (Claude Code, Cursor, Gemini, Copilot).

Detailed conventions live in `.claude/skills/authorizer-*/SKILL.md` and load on demand when matching files are touched. Role-specific expertise lives in `.claude/agents/`.

## What is Authorizer

Open-source, self-hosted authentication and authorization server. Supports 13+ databases, OAuth2/OIDC, social logins, MFA, magic links, role-based access, webhooks, and email templating.

**Stack**: Go 1.24+, Gin, gqlgen, GORM, zerolog, Cobra CLI, JWT, OAuth2/OIDC.
**v2**: CLI flags for all config — no `.env` or OS env vars.

## Quick Commands

```bash
make dev                  # Run with SQLite (dev mode)
make build                # Build server binary
make build-app            # Build login UI (web/app)
make build-dashboard      # Build admin UI (web/dashboard)
make generate-graphql     # Regenerate after schema.graphqls change

make test                 # SQLite integration + storage (via TEST_DBS)
make test-sqlite          # SQLite everywhere (no Docker)
make test-all-db          # All 7 databases via Docker

# Single test
go clean --testcache && TEST_DBS="sqlite" go test -p 1 -v -run TestSignup ./internal/integration_tests/
```

## Architecture (Quick Reference)

**Initialization order** (`cmd/root.go`): Config → Storage → MemoryStore → Token → Email/SMS → OAuth → Events → HTTP/GraphQL → Server

**Key paths**:
- GraphQL schema: `internal/graph/schema.graphqls` | Business logic: `internal/graphql/`
- Storage interface: `internal/storage/provider.go` | SQL: `internal/storage/db/sql/` | NoSQL: `mongodb/`, `arangodb/`, `cassandradb/`, `dynamodb/`, `couchbase/`
- OAuth/OIDC endpoints: `internal/http_handlers/`
- Token management: `internal/token/`
- Tests: `internal/integration_tests/`
- Frontend: `web/app/` (user UI) | `web/dashboard/` (admin UI)

**Pattern**: Every subsystem uses `Dependencies` struct + `New()` → `Provider` interface.

## Critical Rules (Top of Mind)

1. **Admin GraphQL ops prefixed with `_`** — not for public use.
2. **Schema changes MUST update all 13+ database providers.**
3. **Run `make generate-graphql`** after editing `schema.graphqls`.
4. **NEVER commit to main** — always use a feature branch (`feat/`, `fix/`, `security/`, `chore/`), push, open a PR. Main must stay deployable.

Detailed rules load via skills (see below) — don't restate them here.

## AI Agents

| Agent | Model | Focus |
|---|---|---|
| `principal-engineer` | opus | Full SDLC: Plan → Execute → Test → Review across Go, storage, GraphQL, HTTP. Use for any change touching >1 subsystem. |
| `security-engineer` | opus | OAuth2/OIDC, JWT, MFA, vulnerability audit. Second-pass on auth-sensitive PRs. |
| `doc-writer` | haiku | API docs, guides, migration docs. |
| `authz-researcher` | opus | Deep, adversarially-verified research on authz standards (OpenFGA, RFC 8693/8707/9728, MCP, CIBA, AuthZEN). Run before designing/building any authz capability. |
| `fga-engineer` | opus | Implements the OpenFGA migration (Wave 1) per specs/FGA_OPENFGA_MIGRATION_PLAN.md (authorizer-docs repo). |
| `delegation-engineer` | opus | Implements the agentic delegation chain (Wave 2) per specs/AGENTIC_DELEGATION_DESIGN.md (authorizer-docs repo). Security-critical. |

## Project Skills (auto-load on matching files)

| Skill | Fires when editing |
|---|---|
| `authorizer-go-conventions` | any `*.go` file |
| `authorizer-graphql` | `internal/graph/`, `internal/graphql/`, `schema.graphqls` |
| `authorizer-storage` | `internal/storage/` (any provider) |
| `authorizer-http-handlers` | `internal/http_handlers/` |
| `authorizer-security` | auth-sensitive code or `security/` branches |
| `authorizer-testing` | any `*_test.go` |
| `authorizer-frontend` | `web/app/`, `web/dashboard/` |
| `openfga-modeling` | FGA engine (`internal/authorization/`), authz models/tuples, `check_permissions`/`list_permissions`/`_fga_*` GraphQL |
| `agentic-auth-standards` | token exchange, delegation, MCP, agent-identity (`internal/token/`, `internal/http_handlers/`) |

## Token Optimization Notes

- Detailed rules live in skills — they only load when relevant files are touched.
- Agent definitions only load when invoked via the `Task` tool.
- Go files auto-formatted on save via hook (no formatting discussion needed).
- Use `Grep`/`Glob` before exploring — avoid speculative file reads.
- Prefer reading specific line ranges over full files.

---

# Andrej Karpathy Behavioral Guidelines

Behavioral guidelines to reduce common LLM coding mistakes.

## 1. Think Before Coding
**Don't assume. Don't hide confusion. Surface tradeoffs.**
Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First
**Minimum code that solves the problem. Nothing speculative.**
- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.
Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes
**Touch only what you must. Clean up only your own mess.**
When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.
When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

## 4. Goal-Driven Execution
**Define success criteria. Loop until verified.**
Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
