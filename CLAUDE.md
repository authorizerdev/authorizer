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

## Token Optimization Notes

- Detailed rules live in skills — they only load when relevant files are touched.
- Agent definitions only load when invoked via the `Task` tool.
- Go files auto-formatted on save via hook (no formatting discussion needed).
- Use `Grep`/`Glob` before exploring — avoid speculative file reads.
- Prefer reading specific line ranges over full files.
