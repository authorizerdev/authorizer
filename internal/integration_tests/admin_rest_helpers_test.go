package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/gateway"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
)

// newAdminRESTServer boots the in-process gRPC server backed by the supplied
// fully-wired test service provider and mounts the grpc-gateway REST surface on
// a gin router, returning the base URL. This is the REST counterpart of
// newAdminClient: admin endpoints under /v1/admin/* are served exactly as in
// production (gateway -> in-process gRPC -> AdminHandler). Use it together with
// initTestSetup so seeded data (seedUser, seedWebhook, ...) is visible.
func newAdminRESTServer(t *testing.T, ts *testSetup) string {
	t.Helper()

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             ts.Logger,
		Config:          ts.Config,
		ServiceProvider: ts.ServiceProvider,
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
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv.URL
}

// adminRESTJSON issues an admin REST call authenticating via the
// x-authorizer-admin-secret header (empty adminSecret sends no auth header, for
// fail-closed assertions; empty body sends no request body, for GET endpoints).
// It decodes the JSON response into out when out is non-nil and returns the HTTP
// status code.
func adminRESTJSON(t *testing.T, baseURL, method, path, adminSecret, body string, out any) int {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseURL)
	if adminSecret != "" {
		req.Header.Set("x-authorizer-admin-secret", adminSecret)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	if out != nil {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(out))
	}
	return resp.StatusCode
}
