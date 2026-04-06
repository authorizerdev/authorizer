package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// parseTestJWTClaims parses a JWT token's claims without validation (for test inspection only).
func parseTestJWTClaims(t *testing.T, tokenString string) jwt.MapClaims {
	t.Helper()
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	require.NoError(t, err)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	return claims
}

// TestCustomAccessTokenScript tests the custom access token script functionality
// including the 5-second execution timeout added for DoS protection.
func TestCustomAccessTokenScript(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		t.Run("should_add_custom_claims_from_script", func(t *testing.T) {
			cfg.CustomAccessTokenScript = `function(user, tokenPayload) {
				return { custom_claim: "hello", user_email: user.email };
			}`
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "custom_script_" + uuid.New().String() + "@authorizer.dev"
			password := "Password@123"

			_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			})
			require.NoError(t, err)

			loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
				Email:    &email,
				Password: password,
			})
			require.NoError(t, err)
			require.NotNil(t, loginRes)
			require.NotNil(t, loginRes.AccessToken)

			// Parse the access token and verify custom claims are present
			claims := parseTestJWTClaims(t, *loginRes.AccessToken)
			assert.Equal(t, "hello", claims["custom_claim"])
			assert.Equal(t, email, claims["user_email"])
		})

		t.Run("should_not_override_reserved_claims", func(t *testing.T) {
			cfg.CustomAccessTokenScript = `function(user, tokenPayload) {
				return { sub: "hacked", iss: "hacked", roles: ["admin"], custom_field: "allowed" };
			}`
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "reserved_claims_" + uuid.New().String() + "@authorizer.dev"
			password := "Password@123"

			_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			})
			require.NoError(t, err)

			loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
				Email:    &email,
				Password: password,
			})
			require.NoError(t, err)
			require.NotNil(t, loginRes)

			claims := parseTestJWTClaims(t, *loginRes.AccessToken)
			// Reserved claims must NOT be overridden
			assert.NotEqual(t, "hacked", claims["sub"])
			assert.NotEqual(t, "hacked", claims["iss"])
			// Roles should NOT be overridden to admin
			roles, ok := claims["roles"].([]interface{})
			if ok {
				for _, r := range roles {
					assert.NotEqual(t, "admin", r, "reserved 'roles' claim must not be overridden by script")
				}
			}
			// Custom (non-reserved) claims should be added
			assert.Equal(t, "allowed", claims["custom_field"])
		})

		t.Run("should_timeout_infinite_loop_script", func(t *testing.T) {
			cfg.CustomAccessTokenScript = `function(user, tokenPayload) {
				while(true) {} // infinite loop — should be killed after 5 seconds
				return { never: "reached" };
			}`
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "timeout_script_" + uuid.New().String() + "@authorizer.dev"
			password := "Password@123"

			_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			})
			require.NoError(t, err)

			// Measure execution time to verify the timeout works
			start := time.Now()

			// Login should still succeed — the timeout is handled gracefully,
			// custom claims are skipped but the token is still created.
			loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
				Email:    &email,
				Password: password,
			})
			elapsed := time.Since(start)

			require.NoError(t, err)
			require.NotNil(t, loginRes)
			require.NotNil(t, loginRes.AccessToken)

			// The token should be valid but without the custom claim from the timed-out script
			claims := parseTestJWTClaims(t, *loginRes.AccessToken)
			assert.Nil(t, claims["never"], "timed-out script claims must not appear in token")
			// Standard claims should still be present
			assert.NotEmpty(t, claims["sub"])
			assert.NotEmpty(t, claims["iss"])

			// Verify the timeout kicked in within a reasonable range (5-8 seconds for the access + id token)
			assert.Less(t, elapsed, 20*time.Second, "login with infinite loop script should complete within 20 seconds (two 5s timeouts + overhead)")
		})

		t.Run("should_handle_script_error_gracefully", func(t *testing.T) {
			cfg.CustomAccessTokenScript = `function(user, tokenPayload) {
				throw new Error("intentional error");
			}`
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "error_script_" + uuid.New().String() + "@authorizer.dev"
			password := "Password@123"

			_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			})
			require.NoError(t, err)

			// Login should still succeed even with a broken script
			loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
				Email:    &email,
				Password: password,
			})
			require.NoError(t, err)
			require.NotNil(t, loginRes)
			require.NotNil(t, loginRes.AccessToken)
		})

		t.Run("should_work_without_custom_script", func(t *testing.T) {
			cfg.CustomAccessTokenScript = ""
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "no_script_" + uuid.New().String() + "@authorizer.dev"
			password := "Password@123"

			_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			})
			require.NoError(t, err)

			loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
				Email:    &email,
				Password: password,
			})
			require.NoError(t, err)
			require.NotNil(t, loginRes)
			require.NotNil(t, loginRes.AccessToken)

			claims := parseTestJWTClaims(t, *loginRes.AccessToken)
			assert.NotEmpty(t, claims["sub"])
			// Ensure no unexpected claims were added
			assert.Nil(t, claims["custom_claim"])
		})

		t.Run("should_have_custom_claims_in_id_token_too", func(t *testing.T) {
			cfg.CustomAccessTokenScript = `function(user, tokenPayload) {
				return { team: "engineering" };
			}`
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)

			email := "id_token_script_" + uuid.New().String() + "@authorizer.dev"
			password := "Password@123"

			_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			})
			require.NoError(t, err)

			loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
				Email:    &email,
				Password: password,
			})
			require.NoError(t, err)
			require.NotNil(t, loginRes)
			require.NotNil(t, loginRes.IDToken)

			// The custom script runs for both access token and ID token
			claims := parseTestJWTClaims(t, *loginRes.IDToken)
			assert.Equal(t, "engineering", claims["team"])
		})
	})
}

// TestClientIDMismatchMetric verifies that client ID mismatch records a security metric.
func TestClientIDMismatchMetric(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)

		router := setupTestRouter(ts)

		t.Run("records_metric_on_client_id_mismatch", func(t *testing.T) {
			// Send request with wrong client ID to /graphql (not dashboard/app)
			body := `{"query":"{ meta { version } }"}`
			w := sendTestRequest(t, router, "POST", "/graphql", body, map[string]string{
				"Content-Type":           "application/json",
				"X-Authorizer-Client-ID": "wrong-client-id",
				"X-Authorizer-URL":       "http://localhost:8080",
				"Origin":                 "http://localhost:3000",
			})

			assert.Equal(t, 400, w.Code)
			assert.Contains(t, w.Body.String(), "invalid_client_id")

			// Check that the security metric was recorded
			metricsBody := getMetricsBody(t, router)
			assert.Contains(t, metricsBody, `authorizer_security_events_total{event="client_id_mismatch",reason="invalid_client_id"}`)
		})

		t.Run("no_metric_for_valid_client_id", func(t *testing.T) {
			body := `{"query":"{ meta { version } }"}`
			w := sendTestRequest(t, router, "POST", "/graphql", body, map[string]string{
				"Content-Type":           "application/json",
				"X-Authorizer-Client-ID": cfg.ClientID,
				"X-Authorizer-URL":       "http://localhost:8080",
				"Origin":                 "http://localhost:3000",
			})

			// Should not be 400
			assert.NotEqual(t, 400, w.Code)
		})

		t.Run("no_metric_for_dashboard_path_mismatch", func(t *testing.T) {
			mark := `authorizer_security_events_total{event="client_id_mismatch",reason="invalid_client_id"}`
			before := prometheusCounterSample(t, getMetricsBody(t, router), mark)
			w := sendTestRequest(t, router, "GET", "/dashboard/", "", map[string]string{
				"X-Authorizer-Client-ID": "wrong-client-id",
			})
			assert.Equal(t, 400, w.Code)
			after := prometheusCounterSample(t, getMetricsBody(t, router), mark)
			assert.Equal(t, before, after, "dashboard path mismatch must not increment client_id_mismatch metric")
		})

		t.Run("records_client_id_header_missing_metric", func(t *testing.T) {
			mark := "authorizer_client_id_header_missing_total"
			before := prometheusCounterSample(t, getMetricsBody(t, router), mark)
			sendTestRequest(t, router, "POST", "/graphql", `{"query":"{ meta { version } }"}`, map[string]string{
				"Content-Type":     "application/json",
				"X-Authorizer-URL": "http://localhost:8080",
				"Origin":           "http://localhost:3000",
			})
			after := prometheusCounterSample(t, getMetricsBody(t, router), mark)
			assert.Greater(t, after, before)
		})
	})
}

// Helper functions for cleaner test code

func setupTestRouter(ts *testSetup) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ts.HttpProvider.CORSMiddleware())
	router.Use(ts.HttpProvider.CSRFMiddleware())
	router.Use(ts.HttpProvider.ClientCheckMiddleware())
	router.Use(ts.HttpProvider.ContextMiddleware())
	router.POST("/graphql", ts.HttpProvider.GraphqlHandler())
	router.GET("/metrics", ts.HttpProvider.MetricsHandler())
	// Dashboard route to test path exclusion
	router.GET("/dashboard/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	return router
}

func sendTestRequest(t *testing.T, router *gin.Engine, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, path, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	router.ServeHTTP(w, req)
	return w
}

func getMetricsBody(t *testing.T, router *gin.Engine) string {
	t.Helper()
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err)
	router.ServeHTTP(w, req)
	return w.Body.String()
}

// prometheusCounterSample parses the numeric value of the first Prometheus text exposition
// sample line for the given metric name prefix (name plus label set, e.g. "foo" or `bar{a="b"}`).
func prometheusCounterSample(t *testing.T, body, namePrefix string) float64 {
	t.Helper()
	prefix := namePrefix + " "
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		valStr := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		v, err := strconv.ParseFloat(valStr, 64)
		require.NoError(t, err)
		return v
	}
	return 0
}
