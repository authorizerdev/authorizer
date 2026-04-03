package http_handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type visitor struct {
	tokens    float64
	lastSeen  time.Time
	maxTokens float64
	rate      float64 // tokens per second
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     float64 // tokens per second
	burst    int     // max tokens
}

func newRateLimiter(requestsPerMinute int, burst int) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     float64(requestsPerMinute) / 60.0,
		burst:    burst,
	}
	// Clean up stale entries every 5 minutes
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:    float64(rl.burst) - 1,
			lastSeen:  now,
			maxTokens: float64(rl.burst),
			rate:      rl.rate,
		}
		return true
	}

	// Add tokens based on elapsed time
	elapsed := now.Sub(v.lastSeen).Seconds()
	v.tokens += elapsed * v.rate
	if v.tokens > v.maxTokens {
		v.tokens = v.maxTokens
	}
	v.lastSeen = now

	if v.tokens >= 1 {
		v.tokens--
		return true
	}
	return false
}

// RateLimitMiddleware returns a gin middleware that limits requests per IP.
// requestsPerMinute: sustained rate, burst: max burst size.
func (h *httpProvider) RateLimitMiddleware() gin.HandlerFunc {
	// 60 requests per minute with burst of 20
	limiter := newRateLimiter(60, 20)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.allow(ip) {
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
