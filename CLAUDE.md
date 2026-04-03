# CLAUDE.md

Project instructions for AI coding tools (Claude Code, Cursor, Gemini, Copilot).
Detailed conventions are in `.claude/rules/` (loaded on-demand per file type) and `.claude/agents/` (role-specific expertise).

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

# Testing (TEST_DBS env var selects databases, default: postgres)
make test                 # Docker Postgres (default)
make test-sqlite          # SQLite in-memory (no Docker)
make test-mongodb         # Docker MongoDB
make test-all-db          # ALL 7 databases (postgres,sqlite,mongodb,arangodb,scylladb,dynamodb,couchbase)

# Single test against specific DBs
go clean --testcache && TEST_DBS="sqlite,postgres" go test -p 1 -v -run TestSignup ./internal/integration_tests/
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

## Critical Rules

1. **Admin GraphQL ops prefixed with `_`** (e.g., `_users`, `_delete_user`) — not for public use
2. **Schema changes must update ALL 13+ database providers**
3. **Run `make generate-graphql`** after editing `schema.graphqls`
4. **Security**: parameterized queries only, `crypto/rand` for tokens, `crypto/subtle` for comparisons, never log secrets
5. **Tests**: integration tests with real DBs, table-driven subtests, testify assertions

## AI Agent Roles

Detailed agent files in `.claude/agents/`. Summary:

| Agent | Focus |
|-------|-------|
| `software-engineer` | Full SDLC: Plan → Execute → Test → Review, git practices, code review, issue management |
| `golang-engineer` | Go idioms, provider pattern, code style, GraphQL + REST API conventions |
| `security-engineer` | Security + auth protocols: OWASP, OAuth2/OIDC, JWT, MFA, vulnerability audit |
| `database-engineer` | Multi-DB consistency across 13+ providers |
| `doc-writer` | API docs, guides, migration docs |

## Token Optimization Notes

- Detailed rules load on-demand via `.claude/rules/` (only when matching files are accessed)
- Agent definitions load only when invoked
- Go files auto-formatted on save via hook (no formatting discussion needed)
- Use `Grep`/`Glob` tools before exploring — avoid unnecessary file reads
- Prefer reading specific line ranges over full files
