package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
)

// TestHealthHandler verifies the /healthz liveness probe endpoint behaviour.
func TestHealthHandler(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
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
	})
}

// TestReadyHandler verifies the /readyz readiness probe endpoint behaviour.
func TestReadyHandler(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
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
	})
}
