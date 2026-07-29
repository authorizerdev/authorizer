package oauth

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
)

func TestGetOAuthConfig_UsesMockBaseURLUnderE2EEnv(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "http://localhost/", nil)

	cfg := &config.Config{
		Env:                constants.E2EEnv,
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
	}
	p, err := New(cfg, &Dependencies{Log: testLogger(t)})
	require.NoError(t, err)

	oauthCfg, err := p.GetOAuthConfig(ctx, constants.AuthRecipeMethodGoogle)
	require.NoError(t, err)
	assert.Equal(t, "http://mock-oauth:4000/google/authorize", oauthCfg.Endpoint.AuthURL)
	assert.Equal(t, "http://mock-oauth:4000/google/token", oauthCfg.Endpoint.TokenURL)
}

func TestGetOAuthConfig_UsesRealEndpointWhenUnset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "http://localhost/", nil)

	cfg := &config.Config{GoogleClientID: "test-client-id", GoogleClientSecret: "test-client-secret"}
	p, err := New(cfg, &Dependencies{Log: testLogger(t)})
	require.NoError(t, err)

	oauthCfg, err := p.GetOAuthConfig(ctx, constants.AuthRecipeMethodGoogle)
	require.NoError(t, err)
	assert.Equal(t, "https://accounts.google.com/o/oauth2/auth", oauthCfg.Endpoint.AuthURL)
}

// TestGetOAuthConfig_TestEnv_StillUsesRealEndpoint proves E2EEnv and TestEnv
// are genuinely distinct: internal/integration_tests runs with Env=TestEnv
// and must not accidentally start routing social OAuth through the
// e2e-playground mock.
func TestGetOAuthConfig_TestEnv_StillUsesRealEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "http://localhost/", nil)

	cfg := &config.Config{
		Env:                constants.TestEnv,
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
	}
	p, err := New(cfg, &Dependencies{Log: testLogger(t)})
	require.NoError(t, err)

	oauthCfg, err := p.GetOAuthConfig(ctx, constants.AuthRecipeMethodGoogle)
	require.NoError(t, err)
	assert.Equal(t, "https://accounts.google.com/o/oauth2/auth", oauthCfg.Endpoint.AuthURL)
}

func testLogger(t *testing.T) *zerolog.Logger {
	t.Helper()
	l := zerolog.Nop()
	return &l
}
