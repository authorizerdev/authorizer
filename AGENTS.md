# AGENTS.md

Project instructions for AI coding tools (Claude Code, Cursor, Gemini, Copilot).

Detailed conventions live in `.claude/skills/authorizer-*/SKILL.md` and load on demand when matching files are touched. Role-specific expertise lives in `.claude/agents/`.

## What is Authorizer

Open-source, self-hosted authentication and authorization server. Supports 13+ databases, OAuth2/OIDC, social logins, MFA, magic links, role-based access, webhooks, and email templating.

**Stack**: Go 1.24+, Gin, gqlgen, GORM, zerolog, Cobra CLI, JWT, OAuth2/OIDC, gRPC (+ grpc-gateway REST), buf/protobuf.
**v2**: CLI flags for all config — no `.env` or OS env vars.

## Make Commands

### Dev & build

```bash
make dev                  # Run server locally (SQLite, embedded dev RSA keys)
make build                # Cross-compile server binary → build/{os}/{arch}/authorizer
make build-app            # Build login UI (web/app)
make build-dashboard      # Build admin UI (web/dashboard)
make all                  # build + build-app + build-dashboard
make bootstrap            # Install gox (required by make build)
make clean                # Remove build/
```

### Docker

```bash
make build-local-image    # docker build (IMAGE defaults to quay.io/authorizer/authorizer:$(VERSION))
make build-push-image     # Multi-arch buildx push
make trivy-scan           # Scan Docker image for HIGH/CRITICAL CVEs (IMAGE= override)
```

`VERSION` defaults to `0.1.0-local`; override with `make build VERSION=1.2.3`.

### Code generation

```bash
make generate-graphql     # Regenerate gqlgen output after schema.graphqls change; runs go mod tidy
make generate-db-template # Scaffold new storage provider: make generate-db-template dbname=foo

make proto-gen            # buf generate → gen/ (installs buf if missing)
make proto-lint           # buf lint on proto/
make proto-breaking       # Breaking-change check vs origin/main (override: BUF_BREAKING_AGAINST)
make proto-check          # proto-gen + fail if gen/ is stale (CI)
make proto-tools          # Install buf only
```

After editing `internal/graph/schema.graphqls` → `make generate-graphql`.
After editing `proto/` → `make proto-gen` and commit `gen/`.

### Format & lint

```bash
make fmt                  # fmt-go + fmt-ts
make fmt-go               # gofmt -s (excludes gen/)
make fmt-ts               # Prettier on web/app and web/dashboard

make lint                 # lint-go + lint-ts
make lint-go              # golangci-lint (installs if missing; excludes gen/ via .golangci.yml)
make lint-ts              # Prettier --check on both web apps
make lint-tools           # Install golangci-lint only
```

Run `make fmt` before committing; CI runs `make lint`.

### Tests

```bash
make test                 # Full module test run; TEST_DBS=sqlite (integration tests always SQLite)
make test-sqlite          # Same as test — explicit SQLite-only, no Docker
make test-all-db          # All 7 DBs via Docker (postgres, sqlite, mongodb, arangodb, scylladb, dynamodb, couchbase)
make smoke                # Release e2e smoke tests (build tag `smoke`, 5m timeout)

# Single-DB targets (each spins up Docker, runs tests, tears down)
make test-postgres
make test-mongodb
make test-scylladb
make test-arangodb
make test-dynamodb
make test-couchbase

# Docker helpers for test-all-db
make test-docker-up       # Start all test DB containers + Redis
make test-cleanup         # Remove all test containers
make test-cleanup-postgres | test-cleanup-mongodb | test-cleanup-scylladb
make test-cleanup-arangodb | test-cleanup-dynamodb | test-cleanup-couchbase
```

**Test env vars**:
- `TEST_DBS` — comma-separated list for storage provider tests (e.g. `sqlite`, `postgres,mongodb`). Defaults to all when unset in storage tests.
- `TEST_ENABLE_REDIS=1` — include Redis memory_store tests (skipped by default).

**Single test** (integration tests use SQLite via `getTestConfig()`):

```bash
go clean --testcache && TEST_DBS="sqlite" go test -p 1 -v -run TestSignup ./internal/integration_tests/
```

Use `-p 1` for integration tests (shared state). Storage tests honour `TEST_DBS`.

## Architecture (Quick Reference)

**Initialization order** (`cmd/root.go`): Config → Storage → MemoryStore → Token → Email/SMS → OAuth → Events → HTTP/GraphQL → gRPC → Server

**Key paths**:
- GraphQL schema: `internal/graph/schema.graphqls` | Resolvers: `internal/graphql/` | Business logic: `internal/service/`
- gRPC: `proto/authorizer/v1/` → `gen/` | Server: `internal/grpcsrv/` | REST via grpc-gateway
- gRPC auth docs: `docs/grpc-rest-api-spec.md` (§2.3 client metadata)
- Request principal (gRPC interceptor): `internal/authctx/`
- Storage interface: `internal/storage/provider.go` | SQL: `internal/storage/db/sql/` | NoSQL: `mongodb/`, `arangodb/`, `cassandradb/`, `dynamodb/`, `couchbase/`
- FGA engine: `internal/authorization/`
- OAuth/OIDC endpoints: `internal/http_handlers/`
- Token management: `internal/token/`
- Tests: `internal/integration_tests/` | Release smoke: `internal/e2e/`
- Frontend: `web/app/` (user UI) | `web/dashboard/` (admin UI)

**Transport pattern**: Handlers (GraphQL, gRPC, REST) are thin; shared logic lives in `internal/service/` behind `Provider` / `AdminProvider` interfaces. gRPC uses `transport.MetaFromGRPC` to build `RequestMetadata`; the auth interceptor attaches `authctx.Principal` before handlers run.

**Provider pattern**: Every subsystem uses `Dependencies` struct + `New()` → `Provider` interface.

## Critical Rules (Top of Mind)

1. **Admin GraphQL ops prefixed with `_`** — not for public use. Same for `AuthorizerAdminService` gRPC.
2. **Schema changes MUST update all 13+ database providers.**
3. **Run `make generate-graphql`** after editing `schema.graphqls`.
4. **Run `make proto-gen`** (or `make proto-check`) after editing `proto/`; commit `gen/`.
5. **NEVER commit to main** — always use a feature branch (`feat/`, `fix/`, `security/`, `chore/`), push, open a PR. Main must stay deployable.

Detailed rules load via skills (see below) — don't restate them here.

## AI Agents

| Agent | Model | Focus |
|---|---|---|
| `principal-engineer` | opus | Full SDLC: Plan → Execute → Test → Review across Go, storage, GraphQL, HTTP, gRPC. Use for any change touching >1 subsystem. |
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

# Behavioral Guidelines

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
