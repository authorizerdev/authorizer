# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Authorizer

Open-source, self-hosted authentication and authorization server. Supports 13+ databases, OAuth2/OIDC, social logins, MFA, magic links, role-based access, webhooks, and email templating.

**v2 uses CLI arguments for all configuration** — no `.env` or OS env vars. Pass config via flags (e.g. `--client-id=... --client-secret=...`).

## Build & Run Commands

```bash
# Run locally with SQLite (dev mode)
make dev

# Build server binary (cross-platform via gox)
make build

# Build frontend apps
make build-app        # web/app (end-user login UI)
make build-dashboard  # web/dashboard (admin UI)

# Regenerate GraphQL code after editing internal/graph/schema.graphqls
make generate-graphql

# Run all tests (requires Docker for DB containers)
make test

# Run tests for a specific DB only
make test-mongodb
make test-arangodb
make test-scylladb
make test-dynamodb
make test-couchbase

# Run a single test
go clean --testcache && TEST_DBS="sqlite" go test -p 1 -v -run TestSignup ./internal/integration_tests/

# Run tests against specific DBs without Docker setup
go clean --testcache && TEST_DBS="sqlite,mongodb" go test -p 1 -v ./...

# Generate a new DB provider scaffold
make generate-db-template dbname=mydb
```

## Architecture

### Dependency Injection Pattern
The codebase uses a consistent pattern: each subsystem defines a `Dependencies` struct and a `New()` constructor returning a `Provider` interface. Subsystems are wired together in `cmd/root.go`.

Initialization order in `cmd/root.go`:
1. Config parsing (CLI flags via Cobra)
2. Storage provider (database)
3. Memory store (Redis or in-memory)
4. Token provider (JWT)
5. Email/SMS providers
6. OAuth providers
7. Event system (webhooks)
8. HTTP handlers & GraphQL resolvers
9. Server startup

### GraphQL
- Schema: `internal/graph/schema.graphqls`
- Generated code: `internal/graph/generated/` and `internal/graph/model/`
- Config: `gqlgen.yml`
- Resolver implementations: `internal/graph/` (follow-schema layout)
- Business logic: `internal/graphql/` (mutation/query handler functions called by resolvers)

### Storage Layer
- Interface: `internal/storage/provider.go` — all DB providers implement `storage.Provider`
- Schema structs: `internal/storage/schemas/`
- SQL databases (Postgres, MySQL, SQLite, SQLServer, MariaDB, YugabyteDB, PlanetScale, CockroachDB, LibSQL) share implementation via GORM in `internal/storage/db/sql/`
- NoSQL each have their own package: `mongodb/`, `arangodb/`, `cassandradb/`, `dynamodb/`, `couchbase/`
- Template for new providers: `internal/storage/db/provider_template/`

### Testing
- Integration tests live in `internal/integration_tests/`
- Tests use `TEST_DBS` env var to select which database backends to test against
- Test helper (`test_helper.go`) bootstraps a full test setup with real DB connections, GraphQL provider, HTTP server
- Default test DB is Postgres on port 5434

### Key Internal Packages
- `internal/config/` — Config struct, all server settings
- `internal/constants/` — DB types, auth method enums, token types, webhook event names
- `internal/token/` — JWT generation/validation
- `internal/oauth/` — Social login provider integrations
- `internal/memory_store/` — Session/state storage (Redis or DB-backed)
- `internal/authenticators/` — MFA (TOTP) support
- `internal/http_handlers/` — REST endpoints (OAuth callbacks, token endpoints, well-known)
- `internal/events/` — Webhook event system

### Frontend Apps
- `web/app/` — End-user facing React app (login/signup UI), built with Vite
- `web/dashboard/` — Admin dashboard React app, built with Chakra UI + Vite
- Both use `npm ci && npm run build`
