# Rate Limiting Design

## Overview

Add per-IP rate limiting to Authorizer that works across multiple replicas. Uses Redis when available (distributed sliding window), falls back to in-memory token bucket for single-instance deployments. Configurable via CLI flags with sensible defaults.

## CLI Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--rate-limit-rps` | float64 | 10 | Requests per second per IP |
| `--rate-limit-burst` | int | 20 | Max burst size per IP |

Always enabled. No on/off toggle — if you need to disable, set `--rate-limit-rps` to 0 (middleware becomes a passthrough).

## Config

Add to `internal/config/config.go`:

```go
RateLimitRPS   float64
RateLimitBurst int
```

## Architecture

New package `internal/rate_limit/` following the project's provider pattern.

### Provider Interface

```go
type Provider interface {
    Allow(ctx context.Context, ip string) (bool, error)
}
```

### Factory

```go
func New(cfg *config.Config, deps *Dependencies) (Provider, error)
```

Picks implementation based on available infrastructure:
- **Redis configured** (`cfg.RedisURL != ""`) → `redisProvider` (sliding window counter)
- **Otherwise** → `inMemoryProvider` (`golang.org/x/time/rate`)

### Redis Implementation (`redis.go`)

Sliding window counter using atomic Redis operations:

- Key format: `rate_limit:<ip>`
- Uses `INCR` + `EXPIRE` (atomic via pipeline or Lua script)
- Window = 1 second, limit = `RateLimitRPS`
- Burst handled by allowing up to `RateLimitBurst` in any single window
- Keys auto-expire via Redis TTL (no cleanup goroutine needed)

Requires adding to `RedisClient` interface in `internal/memory_store/redis/provider.go`:
- `Incr(ctx, key) *redis.IntCmd`
- `Expire(ctx, key, expiration) *redis.BoolCmd`

Alternatively, the rate_limit package can accept a redis client directly rather than going through memory_store.

### In-Memory Implementation (`in_memory.go`)

- `sync.Map` of IP → `*rate.Limiter` (from `golang.org/x/time/rate`)
- Each limiter configured with `rate.Limit(cfg.RateLimitRPS)` and `cfg.RateLimitBurst`
- Cleanup goroutine with `context.Context` for graceful shutdown (evicts entries unseen for 10 minutes)
- No global mutex — `sync.Map` handles concurrent access

### Dependencies

```go
type Dependencies struct {
    Log        *zerolog.Logger
    RedisURL   string           // empty = use in-memory
    RedisClient redis.RedisClient // nil = use in-memory
}
```

## Middleware

Rewrite `internal/http_handlers/rate_limit.go`:

```go
func (h *httpProvider) RateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Skip non-auth endpoints
        if isExemptPath(c.Request.URL.Path) {
            c.Next()
            return
        }
        // RPS=0 means disabled
        if h.Config.RateLimitRPS <= 0 {
            c.Next()
            return
        }
        allowed, err := h.Dependencies.RateLimitProvider.Allow(c.Request.Context(), c.ClientIP())
        if err != nil {
            // Log error, allow request (fail-open)
            c.Next()
            return
        }
        if !allowed {
            c.Header("Retry-After", "1")
            c.JSON(429, gin.H{
                "error":             "rate_limit_exceeded",
                "error_description": "Too many requests. Please try again later.",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### Exempt Paths (not rate limited)

These endpoints are infrastructure/static and not auth-related:

| Path | Reason |
|------|--------|
| `/` | Root/info endpoint |
| `/health` | Health check (k8s liveness) |
| `/healthz` | Health check (k8s liveness) |
| `/readyz` | Readiness check (k8s readiness) |
| `/metrics` | Prometheus scrape endpoint |
| `/playground` | GraphQL playground (dev tool) |
| `/.well-known/openid-configuration` | OIDC discovery (cacheable, standards-required) |
| `/.well-known/jwks.json` | JWKS endpoint (cacheable, standards-required) |
| `/app/*` | Static frontend assets (login UI) |
| `/dashboard/*` | Static frontend assets (admin UI) |

### Rate-Limited Paths (auth endpoints)

| Path | Why |
|------|-----|
| `/graphql` | All auth mutations (signup, login, reset password, etc.) |
| `/oauth_login/:provider` | OAuth initiation |
| `/oauth_callback/:provider` | OAuth callback |
| `/verify_email` | Email verification |
| `/authorize` | OAuth2 authorize |
| `/userinfo` | Token-based user info |
| `/logout` | Session termination |
| `/oauth/token` | Token exchange |
| `/oauth/revoke` | Token revocation |

## Wiring

In `cmd/root.go`, after memory store init and before HTTP provider init:

```go
rateLimitProvider, err := rate_limit.New(cfg, &rate_limit.Dependencies{...})
```

Pass `rateLimitProvider` into `http_handlers.Dependencies`.

## Error Behavior

- **Redis down**: Fail-open (allow request, log error). Auth availability > rate limiting.
- **RPS = 0**: Middleware becomes passthrough (effective disable).
- **Retry-After header**: Set to `"1"` (1 second) on 429 responses per RFC 6585.

## Testing

- Unit tests for both Redis and in-memory providers
- Integration test: burst requests from same IP, verify 429 after burst exceeded
- Integration test: verify exempt paths are not rate limited
- Use `runForEachDB` pattern for integration tests

## Dependencies

- `golang.org/x/time/rate` — new (Go extended stdlib, zero-risk)
- `github.com/redis/go-redis/v9` — already in go.mod

## Files to Create/Modify

| File | Action |
|------|--------|
| `internal/rate_limit/provider.go` | Create — interface, factory, dependencies |
| `internal/rate_limit/redis.go` | Create — Redis sliding window |
| `internal/rate_limit/in_memory.go` | Create — x/time/rate wrapper |
| `internal/config/config.go` | Modify — add RateLimitRPS, RateLimitBurst |
| `cmd/root.go` | Modify — add CLI flags, wire provider |
| `internal/http_handlers/provider.go` | Modify — add RateLimitProvider to Dependencies |
| `internal/http_handlers/rate_limit.go` | Rewrite — thin middleware calling provider |
| `internal/server/http_routes.go` | Modify — move rate limit middleware placement |
| `internal/integration_tests/rate_limit_test.go` | Create — integration tests |
