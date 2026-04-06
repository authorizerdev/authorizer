package http_handlers

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/authorizerdev/authorizer/internal/metrics"
)

// MetricsMiddleware records HTTP request count and duration for every request.
func (h *httpProvider) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start).Seconds()
		status := fmt.Sprintf("%d", c.Writer.Status())

		if metrics.SkipHTTPRequestMetrics(path) {
			return
		}

		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// MetricsHandler returns a Gin handler that serves the Prometheus metrics endpoint.
func (h *httpProvider) MetricsHandler() gin.HandlerFunc {
	prometheusHandler := promhttp.Handler()
	return func(c *gin.Context) {
		prometheusHandler.ServeHTTP(c.Writer, c.Request)
	}
}
