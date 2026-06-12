package integration_tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/gateway"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/service"
)

// TestRESTMeta exercises GET /v1/meta through the grpc-gateway. Validates
// that the gateway translates the REST call into an in-process gRPC
// invocation against AuthorizerService.Meta, then renders the response as
// JSON. The wrapped response shape (`{"meta": {...}}`) is intentional:
// every AuthorizerService RPC's response is a thin wrapper around the
// inner type so buf STANDARD's RPC_REQUEST_RESPONSE_UNIQUE lint is satisfied.
func TestRESTMeta(t *testing.T) {
	cfg := getTestConfig()
	cfg.ClientID = "test-client"

	log := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	svc, err := service.New(cfg, &service.Dependencies{Log: &log})
	require.NoError(t, err)

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             &log,
		Config:          cfg,
		ServiceProvider: svc,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	gw, cleanup, err := gateway.Handler(ctx, grpcSrv.GRPCServer())
	require.NoError(t, err)
	t.Cleanup(cleanup)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/v1/*path", gin.WrapH(gw))
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/meta")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var got struct {
		Meta struct {
			ClientID string `json:"client_id"`
			Version  string `json:"version"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, "test-client", got.Meta.ClientID)
	require.NotEmpty(t, got.Meta.Version)
}
