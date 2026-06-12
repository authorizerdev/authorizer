//go:build smoke

// Package e2e holds release smoke tests: black-box checks that build the real
// `authorizer` binary, boot it as a subprocess, and exercise every public API
// surface (GraphQL, REST, gRPC, MCP) end to end — including an authenticated
// fine-grained-authorization decision on each surface.
//
// They are deliberately excluded from the regular unit/integration runs (build
// tag `smoke`) because they compile the binary and bind real ports. Run them
// with:
//
//	make smoke
//
// CI runs them on every release (see .github/workflows/release.yaml).
package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// Fixed credentials for the smoke instance. Test-only values.
const (
	smokeJWTSecret    = "smoke-jwt-secret-0123456789"
	smokeAdminSecret  = "smoke-admin-secret"
	smokeClientID     = "11111111-2222-3333-4444-555555555555"
	smokeClientSecret = "smoke-client-secret"
	smokeUserEmail    = "smoke@test.dev"
	smokeUserPassword = "Smoke-Pass-123!"

	// fgaModelDSL is the minimal OpenFGA model the scenario authorizes
	// against: a user can be a viewer of a document.
	fgaModelDSL = "model\n  schema 1.1\ntype user\ntype document\n  relations\n    define viewer: [user]"
)

// TestReleaseSmoke is the release gate: one scenario, four surfaces.
//
//  1. Build the binary and boot it (sqlite storage; FGA auto-derives onto the
//     same sqlite file so the MCP subprocess can share it later).
//  2. Seed via GraphQL: admin login, FGA model + tuple, user signup.
//  3. Assert the same check_permissions / list_permissions decision on
//     GraphQL, REST, and gRPC, plus REST fail-closed and validation paths.
//  4. Stop the server and drive the `authorizer mcp` stdio subcommand through
//     a real MCP handshake with the minted bearer token.
func TestReleaseSmoke(t *testing.T) {
	bin := buildBinary(t)
	dbPath := filepath.Join(t.TempDir(), "smoke.db")
	httpPort, metricsPort, grpcPort := freePort(t), freePort(t), freePort(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)

	serverArgs := []string{
		"--database-type=sqlite", "--database-url=" + dbPath,
		"--jwt-type=HS256", "--jwt-secret=" + smokeJWTSecret,
		"--admin-secret=" + smokeAdminSecret,
		"--client-id=" + smokeClientID, "--client-secret=" + smokeClientSecret,
		fmt.Sprintf("--http-port=%d", httpPort),
		fmt.Sprintf("--metrics-port=%d", metricsPort),
		fmt.Sprintf("--grpc-port=%d", grpcPort),
	}
	stopServer := startServer(t, bin, serverArgs, baseURL)

	gql := newGraphQLClient(t, baseURL)

	// --- Seed: admin session, FGA model, user, tuple --------------------
	gql.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)
	gql.mutate(t, `mutation { _fga_write_model(params:{dsl:"`+strings.ReplaceAll(fgaModelDSL, "\n", `\n`)+`"}) { id } }`)

	signup := gql.mutate(t, `mutation { signup(params:{email:"`+smokeUserEmail+`", password:"`+smokeUserPassword+`", confirm_password:"`+smokeUserPassword+`"}) { access_token user { id } } }`)
	token := signup["signup"].(map[string]any)["access_token"].(string)
	userID := signup["signup"].(map[string]any)["user"].(map[string]any)["id"].(string)
	require.NotEmpty(t, token)
	require.NotEmpty(t, userID)

	gql.mutate(t, `mutation { _fga_write_tuples(params:{tuples:[{user:"user:`+userID+`", relation:"viewer", object:"document:readme"}]}) { message } }`)

	// --- Surface 1: GraphQL ---------------------------------------------
	t.Run("graphql check_permissions", func(t *testing.T) {
		data := gql.query(t, token,
			`query { check_permissions(params:{checks:[{relation:"viewer", object:"document:readme"},{relation:"viewer", object:"document:secret"}]}) { results { object allowed } } }`)
		results := data["check_permissions"].(map[string]any)["results"].([]any)
		require.Len(t, results, 2)
		assert.True(t, results[0].(map[string]any)["allowed"].(bool), "viewer on document:readme")
		assert.False(t, results[1].(map[string]any)["allowed"].(bool), "viewer on document:secret")
	})

	// --- Surface 2: REST (/v1 via grpc-gateway) -------------------------
	t.Run("rest check_permissions", func(t *testing.T) {
		var out struct {
			Results []struct {
				Object  string `json:"object"`
				Allowed bool   `json:"allowed"`
			} `json:"results"`
		}
		status := restJSON(t, baseURL, "/v1/check_permissions", token,
			`{"checks":[{"relation":"viewer","object":"document:readme"},{"relation":"viewer","object":"document:secret"}]}`, &out)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, out.Results, 2)
		assert.True(t, out.Results[0].Allowed)
		assert.False(t, out.Results[1].Allowed)
	})

	t.Run("rest list_permissions", func(t *testing.T) {
		var out struct {
			Objects   []string `json:"objects"`
			Truncated bool     `json:"truncated"`
		}
		status := restJSON(t, baseURL, "/v1/list_permissions", token, `{}`, &out)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, []string{"document:readme"}, out.Objects)
		assert.False(t, out.Truncated)
	})

	t.Run("rest fail-closed and validation", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		// No auth -> 401 unauthenticated.
		status := restJSON(t, baseURL, "/v1/check_permissions", "",
			`{"checks":[{"relation":"viewer","object":"document:readme"}]}`, &env)
		assert.Equal(t, http.StatusUnauthorized, status)
		assert.Equal(t, "unauthenticated", env.Code)
		// Empty checks -> 400 invalid_argument (protovalidate min_items=1).
		status = restJSON(t, baseURL, "/v1/check_permissions", token, `{"checks":[]}`, &env)
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, "invalid_argument", env.Code)
	})

	// --- Surface 3: gRPC -------------------------------------------------
	t.Run("grpc check_permissions", func(t *testing.T) {
		conn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", grpcPort),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		client := authorizerv1.NewAuthorizerServiceClient(conn)

		// Pure-gRPC callers carry the token plus the authorizer host (the
		// issuer the token was minted with) as metadata.
		ctx := metadata.AppendToOutgoingContext(context.Background(),
			"authorization", "Bearer "+token,
			"x-authorizer-url", baseURL)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		res, err := client.CheckPermissions(ctx, &authorizerv1.CheckPermissionsRequest{
			Checks: []*authorizerv1.PermissionCheckInput{
				{Relation: "viewer", Object: "document:readme"},
				{Relation: "viewer", Object: "document:secret"},
			}})
		require.NoError(t, err)
		require.Len(t, res.Results, 2)
		assert.True(t, res.Results[0].Allowed)
		assert.False(t, res.Results[1].Allowed)

		list, err := client.ListPermissions(ctx, &authorizerv1.ListPermissionsRequest{})
		require.NoError(t, err)
		assert.Equal(t, []string{"document:readme"}, list.Objects)
	})

	// --- Surface 4: MCP (stdio subprocess) -------------------------------
	// The MCP subcommand is a separate process sharing the sqlite store, so
	// stop the server first to avoid two writers on one sqlite file.
	stopServer()

	t.Run("mcp stdio", func(t *testing.T) {
		mcpArgs := []string{"mcp",
			"--database-type=sqlite", "--database-url=" + dbPath,
			"--jwt-type=HS256", "--jwt-secret=" + smokeJWTSecret,
			"--admin-secret=" + smokeAdminSecret,
			"--client-id=" + smokeClientID, "--client-secret=" + smokeClientSecret,
			"--mcp-bearer=" + token,
			"--mcp-authorizer-url=" + baseURL,
		}
		mcp := startMCP(t, bin, mcpArgs)

		init := mcp.call(t, "initialize", map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "release-smoke", "version": "1.0"},
		})
		require.Equal(t, "authorizer", init["serverInfo"].(map[string]any)["name"])
		mcp.notify(t, "notifications/initialized")

		tools := mcp.call(t, "tools/list", nil)
		names := map[string]bool{}
		for _, tool := range tools["tools"].([]any) {
			names[tool.(map[string]any)["name"].(string)] = true
		}
		for _, want := range []string{"meta", "profile", "check_permissions", "list_permissions"} {
			assert.True(t, names[want], "tool %q must be exposed", want)
		}
		assert.False(t, names["permissions"], "legacy permissions tool must be gone")

		check := mcp.toolCall(t, "check_permissions", map[string]any{
			"checks": []any{
				map[string]any{"relation": "viewer", "object": "document:readme"},
				map[string]any{"relation": "viewer", "object": "document:secret"},
			}})
		var checkOut struct {
			Results []struct {
				Allowed bool `json:"allowed"`
			} `json:"results"`
		}
		require.NoError(t, json.Unmarshal([]byte(check), &checkOut))
		require.Len(t, checkOut.Results, 2)
		assert.True(t, checkOut.Results[0].Allowed)
		assert.False(t, checkOut.Results[1].Allowed)

		profile := mcp.toolCall(t, "profile", map[string]any{})
		var profOut struct {
			User struct {
				Email string `json:"email"`
			} `json:"user"`
		}
		require.NoError(t, json.Unmarshal([]byte(profile), &profOut))
		assert.Equal(t, smokeUserEmail, profOut.User.Email)
	})
}

// buildBinary compiles the authorizer binary into a temp dir and returns its
// path. Building from source guarantees the smoke run tests exactly the code
// under release, not a stale artifact.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "authorizer")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build: %s", out)
	return bin
}

// repoRoot resolves the module root (two levels up from internal/e2e).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Dir(filepath.Dir(wd))
}

// freePort reserves an ephemeral TCP port and returns it. The listener is
// closed immediately; the tiny reuse race is acceptable for tests.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

// startServer boots the binary, waits until /v1/meta serves, and returns a
// stop function (also registered as cleanup, safe to call twice).
func startServer(t *testing.T, bin string, args []string, baseURL string) func() {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "server.log")
	logFile, err := os.Create(logPath)
	require.NoError(t, err)

	cmd := exec.Command(bin, args...)
	// The server resolves web assets (web/templates/*) relative to its
	// working directory; run it from the repo root like a real deployment.
	cmd.Dir = repoRoot(t)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	require.NoError(t, cmd.Start())

	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		_ = logFile.Close()
	}
	t.Cleanup(stop)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/v1/meta")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return stop
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	logs, _ := os.ReadFile(logPath)
	t.Fatalf("server did not become ready; log:\n%s", logs)
	return stop
}

// graphQLClient is a minimal GraphQL-over-HTTP client with a cookie jar (the
// admin session is cookie-based) and the Origin header the CSRF middleware
// requires on state-changing requests.
type graphQLClient struct {
	url    string
	client *http.Client
}

func newGraphQLClient(t *testing.T, baseURL string) *graphQLClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &graphQLClient{url: baseURL + "/graphql", client: &http.Client{Jar: jar, Timeout: 15 * time.Second}}
}

func (g *graphQLClient) do(t *testing.T, query, bearer string) map[string]any {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, g.url, strings.NewReader(string(body)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", strings.TrimSuffix(g.url, "/graphql"))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := g.client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var out struct {
		Data   map[string]any `json:"data"`
		Errors []any          `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Empty(t, out.Errors, "graphql errors for %s", query)
	return out.Data
}

func (g *graphQLClient) mutate(t *testing.T, query string) map[string]any {
	return g.do(t, query, "")
}

func (g *graphQLClient) query(t *testing.T, bearer, query string) map[string]any {
	return g.do(t, query, bearer)
}

// restJSON POSTs a JSON body to a /v1 path and decodes the response into out.
// Returns the HTTP status. An empty bearer sends no Authorization header.
func restJSON(t *testing.T, baseURL, path, bearer, body string, out any) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseURL)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(out))
	return resp.StatusCode
}

// mcpProc drives an `authorizer mcp` subprocess over stdio JSON-RPC, the same
// transport an MCP host (Claude Code) uses.
type mcpProc struct {
	cmd    *exec.Cmd
	stdin  *json.Encoder
	stdout *bufio.Scanner
	nextID int
}

func startMCP(t *testing.T, bin string, args []string) *mcpProc {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repoRoot(t)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	return &mcpProc{cmd: cmd, stdin: json.NewEncoder(stdin), stdout: scanner}
}

// call sends one JSON-RPC request and returns the `result` object.
func (m *mcpProc) call(t *testing.T, method string, params any) map[string]any {
	t.Helper()
	m.nextID++
	req := map[string]any{"jsonrpc": "2.0", "id": m.nextID, "method": method}
	if params != nil {
		req["params"] = params
	}
	require.NoError(t, m.stdin.Encode(req))
	require.True(t, m.stdout.Scan(), "mcp server closed stdout (scan err: %v)", m.stdout.Err())
	var resp struct {
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	require.NoError(t, json.Unmarshal(m.stdout.Bytes(), &resp))
	require.Nil(t, resp.Error, "jsonrpc error for %s", method)
	return resp.Result
}

// notify sends a JSON-RPC notification (no response expected).
func (m *mcpProc) notify(t *testing.T, method string) {
	t.Helper()
	require.NoError(t, m.stdin.Encode(map[string]any{"jsonrpc": "2.0", "method": method}))
}

// toolCall invokes tools/call, asserts isError=false, and returns the text
// payload of the first content block.
func (m *mcpProc) toolCall(t *testing.T, name string, args map[string]any) string {
	t.Helper()
	res := m.call(t, "tools/call", map[string]any{"name": name, "arguments": args})
	isErr, _ := res["isError"].(bool)
	content := res["content"].([]any)
	require.NotEmpty(t, content)
	text := content[0].(map[string]any)["text"].(string)
	require.False(t, isErr, "tool %s returned error: %s", name, text)
	return text
}
