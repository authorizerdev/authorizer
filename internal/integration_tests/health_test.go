package integration_tests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/http_handlers"
	"github.com/authorizerdev/authorizer/internal/storage"
)

// failingHealthStorage wraps a real storage provider but fails HealthCheck (for probe tests).
type failingHealthStorage struct {
	storage.Provider
}

func (*failingHealthStorage) HealthCheck(ctx context.Context) error {
	return errors.New("test: forced storage health failure")
}

// TestHealthHandler verifies the /healthz liveness probe endpoint behaviour.
func TestHealthHandler(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/healthz", ts.HttpProvider.HealthHandler())

	t.Run("returns_200_when_storage_is_healthy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/healthz", nil)
		require.NoError(t, err)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, "ok", body["status"], "healthy response must contain status=ok")
	})
}

// TestReadyHandler verifies the /readyz readiness probe endpoint behaviour.
func TestReadyHandler(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/readyz", ts.HttpProvider.ReadyHandler())

	t.Run("returns_200_when_storage_is_ready", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/readyz", nil)
		require.NoError(t, err)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, "ready", body["status"], "readiness response must contain status=ready")
	})
}

// TestHealthHandlersUnhealthyStorage verifies liveness/readiness and DB metrics when HealthCheck fails.
func TestHealthHandlersUnhealthyStorage(t *testing.T) {
	cfg := getTestConfig()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	realStorage, err := storage.New(cfg, &storage.Dependencies{Log: &logger})
	require.NoError(t, err)
	t.Cleanup(func() { _ = realStorage.Close() })

	wrapped := &failingHealthStorage{Provider: realStorage}
	httpProv, err := http_handlers.New(cfg, &http_handlers.Dependencies{
		Log:             &logger,
		StorageProvider: wrapped,
	})
	require.NoError(t, err)

	router := gin.New()
	router.GET("/healthz", httpProv.HealthHandler())
	router.GET("/readyz", httpProv.ReadyHandler())
	router.GET("/metrics", httpProv.MetricsHandler())

	t.Run("healthz_returns_503", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/healthz", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "unhealthy", body["status"])
	})

	t.Run("readyz_returns_503", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/readyz", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "not ready", body["status"])
	})

	t.Run("records_unhealthy_db_check_metric", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/healthz", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusServiceUnavailable, w.Code)

		w2 := httptest.NewRecorder()
		req2, err := http.NewRequest(http.MethodGet, "/metrics", nil)
		require.NoError(t, err)
		router.ServeHTTP(w2, req2)
		assert.Contains(t, w2.Body.String(), `authorizer_db_health_check_total{status="unhealthy"}`)
	})
}
