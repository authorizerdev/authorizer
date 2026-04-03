package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TestMetricsEndpoint verifies the /metrics endpoint serves Prometheus format.
func TestMetricsEndpoint(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)

		router := gin.New()
		router.GET("/metrics", ts.HttpProvider.MetricsHandler())

		t.Run("returns_200_with_prometheus_format", func(t *testing.T) {
			// Trigger some metrics so they appear in output
			metrics.RecordAuthEvent("test", "test")
			metrics.RecordSecurityEvent("test", "test")
			metrics.RecordGraphQLError("test")
			metrics.DBHealthCheckTotal.WithLabelValues("test").Inc()

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			body := w.Body.String()
			// Gauge metrics always appear
			assert.Contains(t, body, "authorizer_active_sessions")
			// Counter/histogram metrics appear after first increment
			assert.Contains(t, body, "authorizer_auth_events_total")
			assert.Contains(t, body, "authorizer_security_events_total")
			assert.Contains(t, body, "authorizer_graphql_errors_total")
			assert.Contains(t, body, "authorizer_db_health_check_total")
		})
	})
}

// TestMetricsMiddleware verifies the HTTP metrics middleware records request count.
func TestMetricsMiddleware(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)

		router := gin.New()
		router.Use(ts.HttpProvider.MetricsMiddleware())
		router.GET("/healthz", ts.HttpProvider.HealthHandler())
		router.GET("/metrics", ts.HttpProvider.MetricsHandler())

		t.Run("records_http_request_metrics", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/healthz", nil)
			require.NoError(t, err)
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			// Check metrics endpoint has recorded it
			w2 := httptest.NewRecorder()
			req2, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w2, req2)

			body := w2.Body.String()
			assert.Contains(t, body, `authorizer_http_requests_total{method="GET",path="/healthz",status="200"}`)
		})
	})
}

// TestDBHealthCheckMetrics verifies health check outcomes are tracked.
func TestDBHealthCheckMetrics(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)

		router := gin.New()
		router.GET("/healthz", ts.HttpProvider.HealthHandler())
		router.GET("/metrics", ts.HttpProvider.MetricsHandler())

		t.Run("records_healthy_db_check", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/healthz", nil)
			require.NoError(t, err)
			router.ServeHTTP(w, req)

			var body map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err)
			assert.Equal(t, "ok", body["status"])

			// Verify metric was recorded
			w2 := httptest.NewRecorder()
			req2, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w2, req2)

			metricsBody := w2.Body.String()
			assert.Contains(t, metricsBody, `authorizer_db_health_check_total{status="healthy"}`)
		})
	})
}

// TestAuthEventMetrics verifies that auth events are recorded in metrics.
func TestAuthEventMetrics(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		router := gin.New()
		router.GET("/metrics", ts.HttpProvider.MetricsHandler())

		email := "metrics_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"

		t.Run("records_login_failure_on_bad_credentials", func(t *testing.T) {
			loginReq := &model.LoginRequest{
				Email:    &email,
				Password: "wrong_password",
			}
			_, err := ts.GraphQLProvider.Login(ctx, loginReq)
			assert.Error(t, err)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w, req)

			body := w.Body.String()
			assert.Contains(t, body, `authorizer_auth_events_total{event="login",status="failure"}`)
			assert.Contains(t, body, `authorizer_security_events_total{event="invalid_credentials"`)
		})

		t.Run("records_signup_and_login_success", func(t *testing.T) {
			signupReq := &model.SignUpRequest{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			}
			res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
			require.NoError(t, err)
			assert.NotNil(t, res)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w, req)
			assert.Contains(t, w.Body.String(), `authorizer_auth_events_total{event="signup",status="success"}`)

			loginReq := &model.LoginRequest{
				Email:    &email,
				Password: password,
			}
			loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
			require.NoError(t, err)
			assert.NotNil(t, loginRes)

			w2 := httptest.NewRecorder()
			req2, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w2, req2)
			assert.Contains(t, w2.Body.String(), `authorizer_auth_events_total{event="login",status="success"}`)
		})
	})
}

// TestGraphQLErrorMetrics verifies that GraphQL errors in 200 responses are captured.
func TestGraphQLErrorMetrics(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)

		router := gin.New()
		router.Use(ts.HttpProvider.ContextMiddleware())
		router.Use(ts.HttpProvider.CORSMiddleware())
		router.POST("/graphql", ts.HttpProvider.GraphqlHandler())
		router.GET("/metrics", ts.HttpProvider.MetricsHandler())

		t.Run("captures_graphql_errors_in_200_responses", func(t *testing.T) {
			body := `{"query":"mutation { login(params: {email: \"nonexistent@test.com\", password: \"wrong\"}) { message } }"}`
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-authorizer-url", "http://localhost:8080")
			req.Header.Set("Origin", "http://localhost:3000")
			router.ServeHTTP(w, req)

			// GraphQL always returns 200 even with errors
			assert.Equal(t, http.StatusOK, w.Code)

			// Check metrics endpoint
			w2 := httptest.NewRecorder()
			req2, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w2, req2)

			metricsBody := w2.Body.String()
			assert.Contains(t, metricsBody, "authorizer_graphql_request_duration_seconds")
		})
	})
}

// TestRecordAuthEventHelpers verifies the helper functions work correctly.
func TestRecordAuthEventHelpers(t *testing.T) {
	t.Run("RecordAuthEvent_increments_counter", func(t *testing.T) {
		metrics.RecordAuthEvent(metrics.EventVerifyEmail, metrics.StatusSuccess)
		metrics.RecordAuthEvent(metrics.EventVerifyOTP, metrics.StatusFailure)
		metrics.RecordSecurityEvent("brute_force", "rate_limit")
	})
}

// TestAdminLoginMetrics verifies admin login records metrics.
func TestAdminLoginMetrics(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		router := gin.New()
		router.GET("/metrics", ts.HttpProvider.MetricsHandler())

		t.Run("records_admin_login_failure", func(t *testing.T) {
			loginReq := &model.AdminLoginRequest{
				AdminSecret: "wrong-secret",
			}
			_, err := ts.GraphQLProvider.AdminLogin(ctx, loginReq)
			assert.Error(t, err)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w, req)

			body := w.Body.String()
			assert.Contains(t, body, `authorizer_auth_events_total{event="admin_login",status="failure"}`)
			assert.Contains(t, body, `authorizer_security_events_total{event="invalid_admin_secret"`)
		})

		t.Run("records_admin_login_success", func(t *testing.T) {
			loginReq := &model.AdminLoginRequest{
				AdminSecret: cfg.AdminSecret,
			}
			res, err := ts.GraphQLProvider.AdminLogin(ctx, loginReq)
			require.NoError(t, err)
			assert.NotNil(t, res)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w, req)

			assert.Contains(t, w.Body.String(), `authorizer_auth_events_total{event="admin_login",status="success"}`)
		})
	})
}

// TestForgotPasswordMetrics verifies forgot password records metrics.
func TestForgotPasswordMetrics(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		router := gin.New()
		router.GET("/metrics", ts.HttpProvider.MetricsHandler())

		t.Run("records_forgot_password_failure_for_nonexistent_user", func(t *testing.T) {
			nonExistentEmail := "nonexistent_metrics@authorizer.dev"
			forgotReq := &model.ForgotPasswordRequest{
				Email: refs.NewStringRef(nonExistentEmail),
			}
			_, err := ts.GraphQLProvider.ForgotPassword(ctx, forgotReq)
			assert.Error(t, err)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
			require.NoError(t, err)
			router.ServeHTTP(w, req)

			assert.Contains(t, w.Body.String(), `authorizer_auth_events_total{event="forgot_password",status="failure"}`)
		})
	})
}
