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
// JSON. The response is the flat Meta object (no wrapper), byte-identical to
// the GraphQL `meta` query response.
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
		TokenProvider:   nil,
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
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var got struct {
		ClientID string `json:"client_id"`
		Version  string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, "test-client", got.ClientID)
	require.NotEmpty(t, got.Version)
}
