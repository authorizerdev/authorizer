package integration_tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
)

func TestRateLimitMiddleware(t *testing.T) {
	cfg := getTestConfig()
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
		router.GET("/.well-known/openid-configuration", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"issuer": "test"})
		})

		exemptPaths := []string{"/health", "/.well-known/openid-configuration"}
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

	ctx := context.Background()

	t.Run("should_allow_up_to_burst", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			allowed, err := provider.Allow(ctx, "1.2.3.4")
			require.NoError(t, err)
			assert.True(t, allowed, "request %d should be allowed", i)
		}
	})

	t.Run("should_deny_after_burst", func(t *testing.T) {
		// Use a fresh IP to get a clean limiter
		for i := 0; i < 3; i++ {
			provider.Allow(ctx, "5.6.7.8")
		}
		allowed, err := provider.Allow(ctx, "5.6.7.8")
		require.NoError(t, err)
		assert.False(t, allowed)
	})
}
