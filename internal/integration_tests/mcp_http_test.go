package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/interceptors"
	"github.com/authorizerdev/authorizer/internal/mcp"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// mcpRouter wires the MCP surface exactly as internal/server does: the auth
// middleware that issues the RFC 9728 challenge, in front of the Streamable HTTP
// handler, dispatching over a bufconn-only gRPC server whose interceptor accepts
// only MCP-audience tokens.
//
// Building the real chain rather than stubbing it is the point — the properties
// under test (a 401 that carries discovery information, an audience boundary
// enforced two layers down) only exist when those pieces are assembled together.
func mcpRouter(t *testing.T, ts *testSetup, cfg *config.Config) http.Handler {
	t.Helper()
	log := zerolog.Nop()

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             &log,
		Config:          cfg,
		ServiceProvider: ts.ServiceProvider,
		TokenProvider:   ts.TokenProvider,
		TokenResolver:   interceptors.MCPTokenResolver(ts.TokenProvider, cfg.MCPResource()),
	})
	require.NoError(t, err)

	mcpSrv, err := mcp.New(&log, grpcSrv.GRPCServer(), mcp.Options{Name: "authorizer", Version: "test"})
	require.NoError(t, err)

	router := gin.New()
	router.Any("/mcp", ts.HttpProvider.MCPAuthMiddleware(), gin.WrapH(mcpSrv.Handler()))
	return router
}

func mcpPost(t *testing.T, router http.Handler, bearer, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	router.ServeHTTP(w, req)
	return w
}

// TestMCPHTTPSurface covers the transport end to end: the challenge that starts
// discovery, the audience boundary, and a tool call actually reaching a handler
// under the caller's own identity.
func TestMCPHTTPSurface(t *testing.T) {
	cfg := getTestConfig()
	cfg.AuthorizerURL = "https://auth.example.com"
	cfg.MCPEnabled = true
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// MCP requires --url, and with it set parsers.GetHost returns the canonical
	// URL for every request regardless of headers. Pinning it here is what makes
	// this test exercise the production shape: the token's `iss` and the audience
	// check both resolve to the configured URL rather than to whatever Host an
	// httptest request happens to carry.
	parsers.SetTrustedURL(cfg.AuthorizerURL)
	t.Cleanup(func() { parsers.SetTrustedURL("") })

	resource := cfg.MCPResource()
	router := mcpRouter(t, ts, cfg)

	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef("mcp_http_" + uuid.NewString() + "@authorizer.dev"),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		Roles:           "user",
	})
	require.NoError(t, err)

	t.Run("no credential returns a challenge that starts discovery", func(t *testing.T) {
		w := mcpPost(t, router, "", initializeRPC)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		challenge := w.Header().Get("WWW-Authenticate")
		assert.Contains(t, challenge, `resource_metadata="https://auth.example.com/.well-known/oauth-protected-resource/mcp"`,
			"RFC 9728 §5.1: this pointer is the ONLY thing telling a fresh client where to authenticate")
		assert.NotContains(t, challenge, "error=",
			"RFC 6750 §3: an error parameter describes a credential that was supplied and rejected, not a first contact")
	})

	t.Run("an ordinary login token is refused with invalid_token", func(t *testing.T) {
		// The audience boundary, from the outside. A token that authenticates
		// GraphQL must not authenticate MCP.
		w := mcpPost(t, router, mintStatefulAccessToken(t, ts, user, ""), initializeRPC)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), `error="invalid_token"`,
			"a supplied-but-wrong credential must say so, and must be a 401 so the client refreshes rather than looping")
	})

	t.Run("a token for another resource server is refused", func(t *testing.T) {
		w := mcpPost(t, router, mintStatefulAccessToken(t, ts, user, "https://other.example.com/mcp"), initializeRPC)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("a correctly-audienced token reaches the tool surface", func(t *testing.T) {
		bearer := mintStatefulAccessToken(t, ts, user, resource)

		w := mcpPost(t, router, bearer, initializeRPC)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		w = mcpPost(t, router, bearer, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var resp struct {
			Result struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			} `json:"result"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.NotEmpty(t, resp.Result.Tools, "the proto-annotated tool set must be discoverable")

		names := make([]string, 0, len(resp.Result.Tools))
		for _, tool := range resp.Result.Tools {
			names = append(names, tool.Name)
		}
		assert.Contains(t, names, "profile",
			"an identity-bearing tool must be exposed, or nothing proves the caller's token reached a handler")
	})

	t.Run("GET is refused once authenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+mintStatefulAccessToken(t, ts, user, resource))
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code,
			"no SSE stream is offered: the main listener's 60s write timeout would sever it")
	})
}

// initializeRPC is the MCP handshake a client sends before anything else.
const initializeRPC = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{` +
	`"protocolVersion":"2025-06-18","capabilities":{},` +
	`"clientInfo":{"name":"test","version":"1"}}}`
