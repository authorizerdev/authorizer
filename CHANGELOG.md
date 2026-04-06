# Changelog

All notable changes to Authorizer will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`--rate-limit-fail-closed`**: when the rate-limit backend returns an error, respond with `503` instead of allowing the request (default remains fail-open).
- **`--metrics-host`**: bind address for the dedicated `/metrics` listener (default `127.0.0.1`). Use `0.0.0.0` when a scraper on another host/pod must reach the metrics port over the network; keep the metrics port off public ingress.

### Changed

- **Prometheus `/metrics`**: always served on a **dedicated** HTTP listener (`--metrics-host`:`--metrics-port`, default `127.0.0.1:8081`). **`--http-port` and `--metrics-port` must differ**; `/metrics` is not registered on the main Gin server.
- **HTTP metrics**: unmatched Gin routes use the fixed path label `unmatched` instead of the raw request URL (prevents cardinality attacks).
- **GraphQL metrics**: the `operation` label is now `anonymous` or `op_<sha256-prefix>` so client-supplied operation names cannot explode time-series cardinality.
- **Health/readiness JSON**: failure responses return a generic `error` string; details remain in server logs.
- **OAuth callback JSON**: generic OAuth-style error body on provider processing failure; details remain in logs.
- **`/playground`** is subject to the same per-IP rate limits as other routes (health and OIDC discovery paths stay exempt). **`/metrics`** is not on the main HTTP router.

### Removed

- **`authorizer_client_id_not_found_total`**: replaced by **`authorizer_client_id_header_missing_total`**, which matches the actual behavior (header omitted, request still allowed). Update dashboards and alerts accordingly.

## [2.2.1-rc.0] - 2026-04-06

Pre-release. See [2.2.1-rc.0](https://github.com/authorizerdev/authorizer/releases/tag/2.2.1-rc.0) on GitHub.

### Added

- **CSRF protection** (middleware).
- **Per-IP rate limiting** with Redis and in-memory backends.
- **GraphQL query complexity limit**.
- **5-second execution timeout** for custom access token scripts.

### Security

- **Crypto**: AES-GCM with HKDF key derivation (replaces AES-CFB); RSA 4096, improved `DecryptRSA` error handling and base64-related naming; `crypto/rand` for HMAC key generation.
- **JWT / tokens**: Verify JWT algorithm in parse keyfunc; safe type assertions for claims; bearer extraction case-sensitivity fix; shorter session and refresh token lifetimes; reserved claim blocklist for custom token scripts.
- **Cookies**: `HttpOnly` on all cookies; reduced cookie max-age; `SameSite` on admin cookie (with broader security-header and CORS credential fixes).
- **OAuth / redirects**: Apple ID token signature verified via OIDC; `redirect_uri` validation hardened against open redirects and wildcard abuse.
- **GraphQL**: SSRF protection for `_test_endpoint`; constant-time admin secret comparison; user enumeration mitigated via generic error messages.
- **HTTP / parsers**: Host header validation to reduce injection risk.
- **Storage / DB**: Parameterized AQL in ArangoDB `UpdateUsers`; Cassandra client TLS verification enabled; GORM `AllowGlobalUpdate` disabled; `DeleteSession` implemented for SQL and ArangoDB.
- **Email / templates**: Explicit TLS `ServerName` for SMTP; `html/template` for email rendering (SSTI mitigation); `template.JS` XSS-related fix.
- **Webhooks**: SSRF protection, HMAC signatures, and response size limits.
- **Data exposure**: Password hash excluded from JSON serialization; JWKS no longer leaks HMAC keys.
- **Operational**: Sanitized errors, panics replaced with errors where appropriate; Dockerfiles hardened (defaults, signals, healthcheck); client ID audit logging and CSRF origin validation tightened.

### Fixed

- GitHub OAuth display name handling and **POST logout** behavior.
- MongoDB driver update and related compilation issues.
- Tests: custom script timeout coverage, client-ID metric behavior, and ArangoDB-related test hardening.

**Full changelog**: [2.2.0...2.2.1-rc.0](https://github.com/authorizerdev/authorizer/compare/2.2.0...2.2.1-rc.0)

## [2.2.0] - 2026-04-03

See [2.2.0](https://github.com/authorizerdev/authorizer/releases/tag/2.2.0) on GitHub.

### Added

- **Prometheus metrics**, **health** checks, and **readiness** HTTP endpoints ([#528](https://github.com/authorizerdev/authorizer/pull/528)).

**Full changelog**: [2.1.0...2.2.0](https://github.com/authorizerdev/authorizer/compare/2.1.0...2.2.0)

## [2.1.0] - 2026-04-03

See [2.1.0](https://github.com/authorizerdev/authorizer/releases/tag/2.1.0) on GitHub.

### Added

- **Structured audit logging** system.

### Changed

- **Audit logging** consolidated behind an `internal/audit` provider.

### Security

- **Open redirect**: stricter validation for `redirect_uri`.

**Full changelog**: [2.0.1...2.1.0](https://github.com/authorizerdev/authorizer/compare/2.0.1...2.1.0)

## [2.0.0] - 2025-02-28

### Added

- **CLI-based configuration**: All configuration is now passed at server start via CLI root arguments. No env store in cache or database.
- **New security flags**:
  - `--disable-admin-header-auth`: When `true`, server does not accept `X-Authorizer-Admin-Secret` header; only secure admin cookie is honored. Recommended for production.
  - `--enable-graphql-introspection`: Controls GraphQL introspection on `/graphql` (default `true`; set `false` for hardened production).
- **Metrics endpoint**: Metrics server on port 8081 (configurable via `--metrics-port`).
- **Restructured project layout**:
  - Root-level `main.go` and `cmd/` for CLI
  - `internal/` for core packages (config, graph, storage, etc.)
  - `web/app` and `web/dashboard` for embedded UIs
  - `web/templates` for HTML templates
- **Build outputs**: Binary named `authorizer`; output to `build/<os>/<arch>/authorizer`.
- **Docker improvements**:
  - Multi-arch builds (linux/amd64, linux/arm64)
  - `ENTRYPOINT [ "./authorizer" ]` for passing CLI args at runtime
  - Alpine 3.23 base images
- **Makefile targets**: `make dev`, `make bootstrap`, `make build-local-image`, `make build-push-image`.

### Changed

- **BREAKING**: Configuration is no longer read from `.env` or OS environment variables. Pass config via CLI flags.
- **BREAKING**: `--client-id` and `--client-secret` are **required**; server exits if missing.
- **BREAKING**: Deprecated mutations `_admin_signup`, `_update_env`, `_generate_jwt_keys` now return errors directing users to configure via CLI.
- **BREAKING**: Dashboard cannot update server configuration. Admin secret, JWT keys, and all env must be set at startup.
- **BREAKING**: Flag names use kebab-case (e.g. `--database-url` instead of `database_url`).
- **BREAKING**: Some inverted boolean flags (e.g. `DISABLE_LOGIN_PAGE` → `--enable-login-page` with `false` to disable).
- **BREAKING**: Go version requirement: >= 1.24 (see `go.mod`).
- **BREAKING**: Node.js >= 18 for web app and dashboard builds.
- Database provider template path: `internal/storage/db/provider_template` (was `server/db/providers/provider_template`).
- GraphQL schema and resolvers moved to `internal/graph/`.
- Tests moved to `internal/integration_tests/`; run with `go test -v ./...` from repo root.

### Deprecated

- `database_url`, `database_type`, `log_level`, `redis_url` flags (use kebab-case `--database-url`, etc.).
- `env_file` flag (no longer supported).

### Fixed

- Corrected Makefile `generate-db-template` and DB-specific test targets to use current project structure.
- Docker build and release workflow updated for v2 layout and binary name.

### Migration

See [MIGRATION.md](MIGRATION.md) for a detailed guide from v1 to v2.

---

## [1.x] - Legacy

Authorizer v1 used environment-based configuration stored in cache/DB and configurable via dashboard or `_update_env` mutation. For v1 documentation, see [docs.authorizer.dev](https://docs.authorizer.dev/) and the [v1 release branch](https://github.com/authorizerdev/authorizer).
