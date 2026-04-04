# Rate Limiting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-IP rate limiting that works across multiple replicas (Redis) with in-memory fallback, configurable via `--rate-limit-rps` and `--rate-limit-burst` CLI flags.

**Architecture:** New `internal/rate_limit/` package following the project's provider pattern. Redis provider uses a Lua sliding-window script for atomic distributed counting. In-memory provider uses `golang.org/x/time/rate`. The middleware in `http_handlers` calls the provider and skips non-auth infrastructure paths.

**Tech Stack:** Go 1.24+, `golang.org/x/time/rate`, `github.com/redis/go-redis/v9` (already in go.mod), Gin middleware.

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/rate_limit/provider.go` | Create | Provider interface, Dependencies, factory `New()` |
| `internal/rate_limit/in_memory.go` | Create | In-memory rate limiter using `x/time/rate` + `sync.Map` |
| `internal/rate_limit/redis.go` | Create | Redis sliding-window rate limiter using Lua script |
| `internal/config/config.go` | Modify | Add `RateLimitRPS`, `RateLimitBurst` fields |
| `cmd/root.go` | Modify | Add CLI flags + defaults, wire rate_limit provider |
| `internal/http_handlers/provider.go` | Modify | Add `RateLimitProvider` to Dependencies, add to Provider interface |
| `internal/http_handlers/rate_limit.go` | Rewrite | Thin middleware calling rate_limit.Provider |
| `internal/server/http_routes.go` | Modify | Wire RateLimitMiddleware on auth-only routes |
| `internal/memory_store/redis/provider.go` | Modify | Expose Redis client for rate_limit package |
| `internal/integration_tests/rate_limit_test.go` | Create | Integration tests |

---

### Task 1: Add Config Fields and CLI Flags

**Files:**
- Modify: `internal/config/config.go:250` (end of struct)
- Modify: `cmd/root.go:31-53` (defaults) and `cmd/root.go:67-211` (init flags)

- [ ] **Step 1: Add config fields**

In `internal/config/config.go`, add before the closing brace of the `Config` struct (line 250):

```go
	// Rate Limiting
	// RateLimitRPS is the maximum requests per second per IP
	RateLimitRPS float64
	// RateLimitBurst is the maximum burst size per IP
	RateLimitBurst int
```

- [ ] **Step 2: Add defaults in cmd/root.go**

In `cmd/root.go`, add to the defaults block (after line 52, after `defaultDiscordScopes`):

```go
	defaultRateLimitRPS   = float64(10)
	defaultRateLimitBurst = 20
```

- [ ] **Step 3: Add CLI flags in cmd/root.go init()**

In `cmd/root.go`, add after the cookies flags section (after line 153, after `disable-admin-header-auth`):

```go
	// Rate limiting flags
	f.Float64Var(&rootArgs.config.RateLimitRPS, "rate-limit-rps", defaultRateLimitRPS, "Maximum requests per second per IP for rate limiting")
	f.IntVar(&rootArgs.config.RateLimitBurst, "rate-limit-burst", defaultRateLimitBurst, "Maximum burst size per IP for rate limiting")
```

- [ ] **Step 4: Add defaults in applyFlagDefaults()**

In `cmd/root.go`, add at the end of `applyFlagDefaults()` (before the closing brace around line 288):

```go
	if c.RateLimitRPS == 0 {
		c.RateLimitRPS = defaultRateLimitRPS
	}
	if c.RateLimitBurst == 0 {
		c.RateLimitBurst = defaultRateLimitBurst
	}
```

- [ ] **Step 5: Verify it compiles**

Run: `cd /Users/lakhansamani/personal/authorizer/authorizer && go build ./...`
Expected: Build succeeds

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go cmd/root.go
git commit -m "feat(config): add rate-limit-rps and rate-limit-burst CLI flags"
```

---

### Task 2: Create Rate Limit Provider Interface and In-Memory Implementation

**Files:**
- Create: `internal/rate_limit/provider.go`
- Create: `internal/rate_limit/in_memory.go`

- [ ] **Step 1: Create provider.go with interface and factory**

Create `internal/rate_limit/provider.go`:

```go
package rate_limit

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
)

// RedisClient is any Redis client that supports Eval (e.g., *redis.Client, *redis.ClusterClient)
type RedisClient = redis.Cmdable

// Provider defines the rate limiting interface
type Provider interface {
	// Allow checks if a request from the given IP should be allowed
	Allow(ctx context.Context, ip string) (bool, error)
	// Close cleans up resources used by the provider
	Close() error
}

// Dependencies for rate limit provider
type Dependencies struct {
	Log        *zerolog.Logger
	RedisStore RedisClient
}

// New creates a new rate limit provider based on available infrastructure.
// Uses Redis when RedisStore is provided, falls back to in-memory.
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	if deps.RedisStore != nil {
		return newRedisProvider(cfg, deps)
	}
	return newInMemoryProvider(cfg, deps)
}
```

- [ ] **Step 2: Create in_memory.go**

Create `internal/rate_limit/in_memory.go`:

```go
package rate_limit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/authorizerdev/authorizer/internal/config"
)

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type inMemoryProvider struct {
	visitors sync.Map
	rps      rate.Limit
	burst    int
	cancel   context.CancelFunc
}

func newInMemoryProvider(cfg *config.Config, deps *Dependencies) (*inMemoryProvider, error) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &inMemoryProvider{
		rps:    rate.Limit(cfg.RateLimitRPS),
		burst:  cfg.RateLimitBurst,
		cancel: cancel,
	}
	go p.cleanup(ctx)
	return p, nil
}

// Allow checks if a request from the given IP is allowed
func (p *inMemoryProvider) Allow(_ context.Context, ip string) (bool, error) {
	v, loaded := p.visitors.LoadOrStore(ip, &entry{
		limiter:  rate.NewLimiter(p.rps, p.burst),
		lastSeen: time.Now(),
	})
	e := v.(*entry)
	e.lastSeen = time.Now()
	if loaded {
		p.visitors.Store(ip, e)
	}
	return e.limiter.Allow(), nil
}

// Close stops the cleanup goroutine
func (p *inMemoryProvider) Close() error {
	p.cancel()
	return nil
}

// cleanup removes stale entries every 5 minutes
func (p *inMemoryProvider) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.visitors.Range(func(key, value any) bool {
				e := value.(*entry)
				if time.Since(e.lastSeen) > 10*time.Minute {
					p.visitors.Delete(key)
				}
				return true
			})
		}
	}
}
```

- [ ] **Step 3: Add golang.org/x/time dependency**

Run: `cd /Users/lakhansamani/personal/authorizer/authorizer && go get golang.org/x/time/rate`

- [ ] **Step 4: Verify it compiles**

Run: `go build ./internal/rate_limit/...`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add internal/rate_limit/provider.go internal/rate_limit/in_memory.go go.mod go.sum
git commit -m "feat(rate-limit): add provider interface and in-memory implementation"
```

---

### Task 3: Create Redis Rate Limit Implementation

**Files:**
- Create: `internal/rate_limit/redis.go`

- [ ] **Step 1: Create redis.go**

Create `internal/rate_limit/redis.go`:

```go
package rate_limit

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Lua script for atomic sliding window rate limiting.
// Returns 1 if allowed, 0 if denied.
// KEYS[1] = rate limit key
// ARGV[1] = max requests (burst)
// ARGV[2] = window size in seconds
var rateLimitScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local current = tonumber(redis.call("GET", key) or "0")
if current >= limit then
    return 0
end
current = redis.call("INCR", key)
if current == 1 then
    redis.call("EXPIRE", key, window)
end
if current > limit then
    return 0
end
return 1
`

type redisProvider struct {
	client RedisClient
	burst  int
	window int
	log    *zerolog.Logger
}

func newRedisProvider(cfg *config.Config, deps *Dependencies) (*redisProvider, error) {
	// Window = burst / rps, minimum 1 second
	window := 1
	if cfg.RateLimitRPS > 0 {
		w := float64(cfg.RateLimitBurst) / cfg.RateLimitRPS
		if w > 1 {
			window = int(w)
		}
	}
	return &redisProvider{
		client: deps.RedisStore,
		burst:  cfg.RateLimitBurst,
		window: window,
		log:    deps.Log,
	}, nil
}

// Allow checks if a request from the given IP is allowed using Redis
func (p *redisProvider) Allow(ctx context.Context, ip string) (bool, error) {
	key := "rate_limit:" + ip
	result, err := p.client.Eval(ctx, rateLimitScript, []string{key}, p.burst, p.window).Int64()
	if err != nil {
		p.log.Error().Err(err).Str("ip", ip).Msg("rate limit redis error, failing open")
		return true, nil
	}
	return result == 1, nil
}

// Close is a no-op for Redis provider (Redis client lifecycle managed elsewhere)
func (p *redisProvider) Close() error {
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/rate_limit/...`
Expected: Build succeeds (redis.go references `deps.Log` from Dependencies which has `Log *zerolog.Logger`, already defined in provider.go)

- [ ] **Step 3: Commit**

```bash
git add internal/rate_limit/redis.go
git commit -m "feat(rate-limit): add Redis sliding-window implementation"
```

---

### Task 4: Wire Rate Limit Provider into HTTP Handlers

**Files:**
- Modify: `internal/http_handlers/provider.go:20-42` (Dependencies struct) and `63-112` (Provider interface)
- Rewrite: `internal/http_handlers/rate_limit.go`

- [ ] **Step 1: Add RateLimitProvider to Dependencies**

In `internal/http_handlers/provider.go`, add the import:

```go
	"github.com/authorizerdev/authorizer/internal/rate_limit"
```

Add to the `Dependencies` struct (after OAuthProvider, line 41):

```go
	// RateLimitProvider is used for per-IP rate limiting
	RateLimitProvider rate_limit.Provider
```

- [ ] **Step 2: Add RateLimitMiddleware to Provider interface**

In `internal/http_handlers/provider.go`, add to the `Provider` interface (after `LoggerMiddleware`, line 107):

```go
	// RateLimitMiddleware is the middleware that rate limits requests per IP
	RateLimitMiddleware() gin.HandlerFunc
```

- [ ] **Step 3: Rewrite rate_limit.go**

Replace the entire contents of `internal/http_handlers/rate_limit.go` with:

```go
package http_handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// exemptPrefixes are path prefixes that bypass rate limiting.
// These are infrastructure, static asset, and OIDC discovery endpoints.
var exemptPrefixes = []string{
	"/app/",
	"/dashboard/",
}

// exemptPaths are exact paths that bypass rate limiting.
var exemptPaths = map[string]bool{
	"/":                                  true,
	"/health":                            true,
	"/healthz":                           true,
	"/readyz":                            true,
	"/metrics":                           true,
	"/playground":                        true,
	"/.well-known/openid-configuration": true,
	"/.well-known/jwks.json":            true,
}

func isExemptPath(path string) bool {
	if exemptPaths[path] {
		return true
	}
	for _, prefix := range exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// RateLimitMiddleware returns a gin middleware that rate limits requests per IP.
func (h *httpProvider) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isExemptPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		if h.Config.RateLimitRPS <= 0 {
			c.Next()
			return
		}

		if h.Dependencies.RateLimitProvider == nil {
			c.Next()
			return
		}

		allowed, err := h.Dependencies.RateLimitProvider.Allow(c.Request.Context(), c.ClientIP())
		if err != nil {
			log := h.Dependencies.Log.With().Str("func", "RateLimitMiddleware").Logger()
			log.Error().Err(err).Msg("rate limit check failed, allowing request")
			c.Next()
			return
		}

		if !allowed {
			c.Header("Retry-After", "1")
			c.JSON(http.StatusTooManyRequests, gin.H{
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

- [ ] **Step 4: Verify it compiles**

Run: `go build ./internal/http_handlers/...`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add internal/http_handlers/provider.go internal/http_handlers/rate_limit.go
git commit -m "feat(http): rewrite rate limit middleware to use provider pattern"
```

---

### Task 5: Wire Rate Limit into Server Routes and cmd/root.go

**Files:**
- Modify: `internal/server/http_routes.go:18-19`
- Modify: `cmd/root.go:363-424` (provider wiring section)
- Modify: `internal/rate_limit/provider.go` (use `redis.Cmdable` for RedisClient)
- Modify: `internal/memory_store/redis/provider.go` (add `Eval` to RedisClient interface, add `Client()` method)

**Approach:** The memory_store redis provider already holds a `RedisClient`. We add `Eval` to that interface (which `*redis.Client` and `*redis.ClusterClient` already satisfy), expose it via `Client()`, and type-assert in `cmd/root.go` to pass it to the rate_limit provider. This reuses the existing Redis connection — no duplicate connections.

- [ ] **Step 1: Add `Eval` to memory_store redis `RedisClient` interface**

In `internal/memory_store/redis/provider.go`, add to the `RedisClient` interface (after the `Keys` method, line 35):

```go
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
```

- [ ] **Step 2: Add `Client()` method to redis memory store provider**

In `internal/memory_store/redis/provider.go`, add after `NewRedisProvider` (after line 94):

```go
// Client returns the underlying Redis client
func (p *provider) Client() RedisClient {
	return p.store
}
```

- [ ] **Step 3: Add rate limit middleware in http_routes.go**

In `internal/server/http_routes.go`, add the rate limit middleware after the CORS middleware (line 18) and before `ClientCheckMiddleware` (line 19):

```go
	router.Use(s.Dependencies.HTTPProvider.RateLimitMiddleware())
```

So lines 18-20 become:

```go
	router.Use(s.Dependencies.HTTPProvider.CORSMiddleware())
	router.Use(s.Dependencies.HTTPProvider.RateLimitMiddleware())
	router.Use(s.Dependencies.HTTPProvider.ClientCheckMiddleware())
```

- [ ] **Step 4: Wire rate limit provider in cmd/root.go**

In `cmd/root.go`, add the import:

```go
	"github.com/authorizerdev/authorizer/internal/rate_limit"
```

After memory store provider creation (after line 370, before sms provider on line 373), add:

```go
	// Rate limit provider
	rateLimitDeps := &rate_limit.Dependencies{
		Log: &log,
	}
	// If memory store is Redis-backed, reuse its client for distributed rate limiting
	type redisClientProvider interface {
		Client() interface {
			Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
		}
	}
	if rcp, ok := memoryStoreProvider.(redisClientProvider); ok {
		if client, ok := rcp.Client().(rate_limit.RedisClient); ok {
			rateLimitDeps.RedisStore = client
		}
	}
	rateLimitProvider, err := rate_limit.New(&rootArgs.config, rateLimitDeps)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create rate limit provider")
	}
```

Note: This uses a local interface `redisClientProvider` for the type assertion, so `cmd/root.go` does not need to import the redis memory store package directly. The go-redis `*redis.Client` and `*redis.ClusterClient` both satisfy `redis.Cmdable` (which is `rate_limit.RedisClient`), so the second assertion always succeeds when Redis is in use.

Also add the go-redis import for the `Cmd` type used in the interface:

```go
	redis "github.com/redis/go-redis/v9"
```

Then add `RateLimitProvider` to the `httpProvider` creation (line 410-421). Add it after `OAuthProvider`:

```go
		RateLimitProvider: rateLimitProvider,
```

- [ ] **Step 5: Verify it compiles**

Run: `go build ./...`
Expected: Build succeeds

- [ ] **Step 6: Commit**

```bash
git add internal/server/http_routes.go cmd/root.go internal/memory_store/redis/provider.go
git commit -m "feat(server): wire rate limit provider into HTTP middleware and CLI"
```

---

### Task 6: Write Integration Tests

**Files:**
- Create: `internal/integration_tests/rate_limit_test.go`

- [ ] **Step 1: Create rate_limit_test.go**

Create `internal/integration_tests/rate_limit_test.go`:

```go
package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
)

func TestRateLimitMiddleware(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		// Set low rate limit for testing
		cfg.RateLimitRPS = 5
		cfg.RateLimitBurst = 5

		ts := initTestSetup(t, cfg)

		t.Run("should_allow_requests_within_limit", func(t *testing.T) {
			w := httptest.NewRecorder()
			_, router := gin.CreateTestContext(w)
			router.Use(ts.HttpProvider.RateLimitMiddleware())
			router.POST("/graphql", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"data": "ok"})
			})

			// First request should succeed
			req, err := http.NewRequest(http.MethodPost, "/graphql", nil)
			require.NoError(t, err)
			req.RemoteAddr = "192.168.1.1:1234"

			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("should_reject_requests_over_limit", func(t *testing.T) {
			w := httptest.NewRecorder()
			_, router := gin.CreateTestContext(w)
			router.Use(ts.HttpProvider.RateLimitMiddleware())
			router.POST("/graphql", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"data": "ok"})
			})

			// Exhaust the burst
			for i := 0; i < 5; i++ {
				req, err := http.NewRequest(http.MethodPost, "/graphql", nil)
				require.NoError(t, err)
				req.RemoteAddr = "10.0.0.1:1234"
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
			}

			// Next request should be rejected
			req, err := http.NewRequest(http.MethodPost, "/graphql", nil)
			require.NoError(t, err)
			req.RemoteAddr = "10.0.0.1:1234"
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
			assert.Contains(t, w.Header().Get("Retry-After"), "1")
		})

		t.Run("should_not_rate_limit_exempt_paths", func(t *testing.T) {
			w := httptest.NewRecorder()
			_, router := gin.CreateTestContext(w)
			router.Use(ts.HttpProvider.RateLimitMiddleware())
			router.GET("/health", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
			router.GET("/metrics", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
			router.GET("/.well-known/openid-configuration", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"issuer": "test"})
			})

			exemptPaths := []string{"/health", "/metrics", "/.well-known/openid-configuration"}
			for _, path := range exemptPaths {
				// Make many requests - none should be limited
				for i := 0; i < 10; i++ {
					req, err := http.NewRequest(http.MethodGet, path, nil)
					require.NoError(t, err)
					req.RemoteAddr = "10.0.0.2:1234"
					w = httptest.NewRecorder()
					router.ServeHTTP(w, req)
					assert.Equal(t, http.StatusOK, w.Code, "path %s request %d should not be rate limited", path, i)
				}
			}
		})

		t.Run("should_isolate_rate_limits_per_ip", func(t *testing.T) {
			w := httptest.NewRecorder()
			_, router := gin.CreateTestContext(w)
			router.Use(ts.HttpProvider.RateLimitMiddleware())
			router.POST("/graphql", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"data": "ok"})
			})

			// Exhaust burst for IP A
			for i := 0; i < 5; i++ {
				req, err := http.NewRequest(http.MethodPost, "/graphql", nil)
				require.NoError(t, err)
				req.RemoteAddr = "10.0.0.3:1234"
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}

			// IP B should still be allowed
			req, err := http.NewRequest(http.MethodPost, "/graphql", nil)
			require.NoError(t, err)
			req.RemoteAddr = "10.0.0.4:1234"
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("should_return_correct_error_format", func(t *testing.T) {
			w := httptest.NewRecorder()
			_, router := gin.CreateTestContext(w)
			router.Use(ts.HttpProvider.RateLimitMiddleware())
			router.POST("/graphql", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"data": "ok"})
			})

			// Exhaust the burst
			for i := 0; i < 6; i++ {
				req, err := http.NewRequest(http.MethodPost, "/graphql", nil)
				require.NoError(t, err)
				req.RemoteAddr = "10.0.0.5:1234"
				w = httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}

			// Check the 429 response body has OAuth2 error format
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
			assert.Contains(t, w.Body.String(), "rate_limit_exceeded")
			assert.Contains(t, w.Body.String(), "error_description")
		})
	})
}

func TestInMemoryRateLimitProvider(t *testing.T) {
	cfg := &config.Config{
		RateLimitRPS:   5,
		RateLimitBurst: 3,
	}
	logger := zerolog.New(zerolog.NewTestWriter(t))
	provider, err := rate_limit.New(cfg, &rate_limit.Dependencies{
		Log: &logger,
	})
	require.NoError(t, err)
	defer provider.Close()

	t.Run("should_allow_up_to_burst", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			allowed, err := provider.Allow(t.Context(), "1.2.3.4")
			require.NoError(t, err)
			assert.True(t, allowed, "request %d should be allowed", i)
		}
	})

	t.Run("should_deny_after_burst", func(t *testing.T) {
		// Use a fresh IP to get a clean limiter
		for i := 0; i < 3; i++ {
			provider.Allow(t.Context(), "5.6.7.8")
		}
		allowed, err := provider.Allow(t.Context(), "5.6.7.8")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}
```

- [ ] **Step 2: Add zerolog import**

Make sure the test file has the zerolog import:

```go
	"github.com/rs/zerolog"
```

- [ ] **Step 3: Update initTestSetup to wire rate limit provider**

In `internal/integration_tests/test_helper.go`, add the rate_limit import:

```go
	"github.com/authorizerdev/authorizer/internal/rate_limit"
```

After the token provider initialization (after line 234) and before the audit provider (line 237), add:

```go
	rateLimitProvider, err := rate_limit.New(cfg, &rate_limit.Dependencies{
		Log: &logger,
	})
	require.NoError(t, err)
```

Then add `RateLimitProvider: rateLimitProvider` to the `httpDeps` struct (after `TokenProvider` on line 265):

```go
		RateLimitProvider: rateLimitProvider,
```

- [ ] **Step 4: Run tests with SQLite**

Run: `cd /Users/lakhansamani/personal/authorizer/authorizer && TEST_DBS=sqlite go test -p 1 -v -run TestRateLimitMiddleware ./internal/integration_tests/`
Expected: All subtests pass

- [ ] **Step 5: Run TestInMemoryRateLimitProvider**

Run: `TEST_DBS=sqlite go test -p 1 -v -run TestInMemoryRateLimitProvider ./internal/integration_tests/`
Expected: All subtests pass

- [ ] **Step 6: Commit**

```bash
git add internal/integration_tests/rate_limit_test.go internal/integration_tests/test_helper.go
git commit -m "test(rate-limit): add integration tests for rate limiting middleware and provider"
```

---

### Task 7: Remove Old Rate Limit Code from Branch

**Files:**
- Verify: No leftover code from the original `rate_limit.go` (visitor struct, rateLimiter struct, etc.)

- [ ] **Step 1: Verify old code is gone**

The rewrite in Task 4 Step 3 replaced the entire file. Verify no old types remain:

Run: `grep -n "type visitor struct\|type rateLimiter struct\|func newRateLimiter" internal/http_handlers/rate_limit.go`
Expected: No matches

- [ ] **Step 2: Run full build**

Run: `go build ./...`
Expected: Build succeeds with no errors

- [ ] **Step 3: Run full test suite with SQLite**

Run: `TEST_DBS=sqlite go test -p 1 -v ./internal/integration_tests/ -count=1`
Expected: All tests pass

- [ ] **Step 4: Commit (if any cleanup needed)**

Only commit if changes were needed during verification.

---

## Summary of Changes

1. **Config**: 2 new fields (`RateLimitRPS`, `RateLimitBurst`) + 2 CLI flags
2. **New package**: `internal/rate_limit/` with provider interface, in-memory, and Redis implementations (3 files)
3. **Middleware**: Thin wrapper calling provider, with exempt path list
4. **Wiring**: `cmd/root.go` creates rate limit provider (Redis if URL set, else in-memory) and passes to HTTP handlers
5. **Tests**: Integration tests covering burst enforcement, exempt paths, per-IP isolation, error format
