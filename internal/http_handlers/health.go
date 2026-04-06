package http_handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/metrics"
)

// HealthHandler is the handler for /healthz liveness probe route.
// It performs a storage health check and returns 200 if healthy or 503 if not.
func (h *httpProvider) HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.Dependencies.StorageProvider.HealthCheck(c.Request.Context()); err != nil {
			h.Dependencies.Log.Error().Err(err).Msg("storage health check failed")
			metrics.DBHealthCheckTotal.WithLabelValues("unhealthy").Inc()
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "storage unavailable",
			})
			return
		}
		metrics.DBHealthCheckTotal.WithLabelValues("healthy").Inc()
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// ReadyHandler is the handler for /readyz readiness probe route.
// It checks storage health and returns 200 if ready or 503 if not.
func (h *httpProvider) ReadyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.Dependencies.StorageProvider.HealthCheck(c.Request.Context()); err != nil {
			h.Dependencies.Log.Error().Err(err).Msg("storage health check failed in readiness probe")
			metrics.DBHealthCheckTotal.WithLabelValues("unhealthy").Inc()
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "storage unavailable",
			})
			return
		}
		metrics.DBHealthCheckTotal.WithLabelValues("healthy").Inc()
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
