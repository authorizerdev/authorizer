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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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
	// `agent` is declared so the RFC 8693 delegation intersection has a subject
	// type to hold the agent half. Declaring the type IS the opt-in; the user
	// assertions elsewhere in this file are unaffected by its presence.
	fgaModelDSL = "model\n  schema 1.1\ntype user\ntype agent\ntype document\n  relations\n    define viewer: [user, agent]"
)

// TestReleaseSmoke is the release gate: one scenario across every public
// surface.
//
//  1. Build the binary and boot it (sqlite storage; FGA auto-derives onto the
//     same sqlite file so the MCP subprocess can share it later).
//  2. Seed via GraphQL: admin login, FGA model + tuple, user signup.
//  3. Assert the same check_permissions / list_permissions decision on
//     GraphQL, REST, and gRPC, plus REST fail-closed and validation paths.
//  4. Run the OAuth 2.1 authorization-code + PKCE round trip — /authorize to
//     /oauth/token to /userinfo, including code single-use. Everything else
//     here authenticates with a token minted directly by signup, so without
//     this the authorization endpoint, the code store, the PKCE comparison and
//     the token endpoint could all break without a single failure.
//  5. Provision a user over inbound SCIM 2.0, the one surface authenticated by
//     a per-org bearer token rather than a session or the admin secret.
//  6. Stop the server and drive the `authorizer mcp` stdio subcommand through
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
		// This scenario exercises FGA permission checks, not MFA. MFA is on by
		// default (TOTP/WebAuthn need no external provider), which would
		// withhold signup's token behind the MFA-setup gate instead of
		// returning it directly.
		"--disable-mfa",
		// MCP over HTTP. --url is mandatory with it: the audience every MCP
		// token is checked against is derived from --url alone, so the binary
		// refuses to start without one.
		"--mcp-enabled",
		"--url=" + baseURL,
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

	// --- Surface 5: Admin (AuthorizerAdminService over REST + gRPC) ------
	// Admin endpoints authenticate via the x-authorizer-admin-secret header
	// (or the admin session cookie), not the bearer token. They are served by
	// the same single gRPC server + REST gateway as the public surface.
	t.Run("rest admin meta", func(t *testing.T) {
		var out struct {
			AdminMeta struct {
				Roles []string `json:"roles"`
			} `json:"admin_meta"`
		}
		// With the admin secret -> 200 + configured roles.
		status := adminREST(t, baseURL, http.MethodGet, "/v1/admin/meta", smokeAdminSecret, "", &out)
		require.Equal(t, http.StatusOK, status)
		assert.NotEmpty(t, out.AdminMeta.Roles)
	})

	t.Run("rest admin users", func(t *testing.T) {
		var out struct {
			Users []struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"users"`
		}
		status := adminREST(t, baseURL, http.MethodPost, "/v1/admin/users", smokeAdminSecret, `{}`, &out)
		require.Equal(t, http.StatusOK, status)
		// The smoke user seeded above must be present.
		var found bool
		for _, u := range out.Users {
			if u.ID == userID {
				found = true
			}
		}
		assert.True(t, found, "seeded smoke user listed via /v1/admin/users")
	})

	t.Run("rest admin fail-closed", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		// No admin secret -> 401 unauthenticated.
		status := adminREST(t, baseURL, http.MethodGet, "/v1/admin/meta", "", "", &env)
		assert.Equal(t, http.StatusUnauthorized, status)
		assert.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("grpc admin meta", func(t *testing.T) {
		conn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", grpcPort),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		client := authorizerv1.NewAuthorizerAdminServiceClient(conn)

		// Fail-closed: no admin secret -> Unauthenticated.
		_, err = client.AdminMeta(context.Background(), &authorizerv1.AdminMetaRequest{})
		require.Error(t, err)

		// With the admin secret as metadata -> roles returned.
		ctx := metadata.AppendToOutgoingContext(context.Background(),
			"x-authorizer-admin-secret", smokeAdminSecret)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		res, err := client.AdminMeta(ctx, &authorizerv1.AdminMetaRequest{})
		require.NoError(t, err)
		require.NotNil(t, res.AdminMeta)
		assert.NotEmpty(t, res.AdminMeta.Roles)
	})

	// --- Surface 6: the OAuth 2.1 authorization code + PKCE round trip ----
	// The flow every browser-based integration depends on, driven end to end
	// against the booted binary: /authorize issues a code, /oauth/token
	// exchanges it with the PKCE verifier, and the resulting access token is
	// accepted at /userinfo.
	//
	// Worth a release gate of its own because everything else here authenticates
	// with a token minted directly by signup — none of it would notice
	// /authorize, the code store, the PKCE comparison or the token endpoint
	// breaking. This is also the only place the flow runs with the real route
	// table, real middleware (CSRF, CORS, rate limiting) and real cookies.
	t.Run("oauth authorization code + PKCE", func(t *testing.T) {
		verifier := "smoke-code-verifier-0123456789abcdefghijklmnop"
		sum := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(sum[:])
		redirectURI := baseURL + "/smoke-callback"

		// Its OWN user and cookie jar, deliberately. /authorize rolls the
		// session over on success, which invalidates the token minted at signup
		// — so running this against the shared user would silently break every
		// later subtest that still holds that token. It did exactly that to the
		// MCP stdio case before this was split out.
		oauthGQL := newGraphQLClient(t, baseURL)
		const oauthEmail = "smoke-oauth@test.dev"
		oauthSignup := oauthGQL.mutate(t, `mutation { signup(params:{email:"`+oauthEmail+`", password:"`+smokeUserPassword+`", confirm_password:"`+smokeUserPassword+`"}) { user { id } } }`)
		oauthUserID := oauthSignup["signup"].(map[string]any)["user"].(map[string]any)["id"].(string)
		require.NotEmpty(t, oauthUserID)

		// The session cookie from that signup is what makes /authorize issue a
		// code without a login round trip; it rides on that client's jar.
		q := url.Values{}
		q.Set("response_type", "code")
		q.Set("client_id", smokeClientID)
		q.Set("redirect_uri", redirectURI)
		q.Set("scope", "openid profile email")
		q.Set("state", "smoke-state")
		q.Set("response_mode", "query")
		q.Set("code_challenge", challenge)
		q.Set("code_challenge_method", "S256")

		req, err := http.NewRequest(http.MethodGet, baseURL+"/authorize?"+q.Encode(), nil)
		require.NoError(t, err)
		// Do NOT follow the redirect: its Location IS the result under test.
		noRedirect := &http.Client{
			Jar:           oauthGQL.client.Jar,
			Timeout:       15 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
		resp, err := noRedirect.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusFound, resp.StatusCode, "authorize must redirect with a code: %s", body)

		loc, err := url.Parse(resp.Header.Get("Location"))
		require.NoError(t, err)
		code := loc.Query().Get("code")
		require.NotEmpty(t, code, "authorize must mint an authorization code")
		assert.Equal(t, "smoke-state", loc.Query().Get("state"), "state must round-trip unmodified")

		// Exchange. The smoke client is confidential, so it authenticates with
		// its secret AND supplies the PKCE verifier.
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("redirect_uri", redirectURI)
		form.Set("client_id", smokeClientID)
		form.Set("client_secret", smokeClientSecret)
		form.Set("code_verifier", verifier)

		tokenResp, err := http.Post(baseURL+"/oauth/token",
			"application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		defer func() { _ = tokenResp.Body.Close() }()
		tokenBody, _ := io.ReadAll(tokenResp.Body)
		require.Equal(t, http.StatusOK, tokenResp.StatusCode, "token exchange failed: %s", tokenBody)

		var tokens struct {
			AccessToken string `json:"access_token"`
			IDToken     string `json:"id_token"`
			TokenType   string `json:"token_type"`
		}
		require.NoError(t, json.Unmarshal(tokenBody, &tokens))
		require.NotEmpty(t, tokens.AccessToken)
		require.NotEmpty(t, tokens.IDToken, "an openid scope must yield an id_token")
		assert.Equal(t, "Bearer", tokens.TokenType)

		// A second exchange of the same code must fail: authorization codes are
		// single-use (RFC 6749 §4.1.2), and a replay is the classic attack.
		replay, err := http.Post(baseURL+"/oauth/token",
			"application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		defer func() { _ = replay.Body.Close() }()
		assert.NotEqual(t, http.StatusOK, replay.StatusCode, "an authorization code must not be redeemable twice")

		// The token works where a real client would use it.
		uiReq, err := http.NewRequest(http.MethodGet, baseURL+"/userinfo", nil)
		require.NoError(t, err)
		uiReq.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		ui, err := http.DefaultClient.Do(uiReq)
		require.NoError(t, err)
		defer func() { _ = ui.Body.Close() }()
		uiBody, _ := io.ReadAll(ui.Body)
		require.Equal(t, http.StatusOK, ui.StatusCode, "userinfo rejected the freshly issued token: %s", uiBody)
		var claims map[string]any
		require.NoError(t, json.Unmarshal(uiBody, &claims))
		assert.Equal(t, oauthUserID, claims["sub"], "userinfo must describe the user who authorized")
	})

	// --- Surface 7: inbound SCIM 2.0 -------------------------------------
	// Provisioning is how enterprise customers create users, and it is the one
	// surface authenticated by a per-org bearer token rather than a session or
	// the admin secret — a route-group or middleware change can break it
	// without touching anything else in this file.
	t.Run("scim provisioning", func(t *testing.T) {
		org := gql.mutate(t, `mutation { _create_organization(params:{name:"smoke-org", display_name:"Smoke Org"}) { id } }`)
		orgID := org["_create_organization"].(map[string]any)["id"].(string)
		require.NotEmpty(t, orgID)

		created := gql.mutate(t, `mutation { _create_scim_endpoint(params:{org_id:"`+orgID+`"}) { token scim_endpoint { id enabled } } }`)
		scimToken := created["_create_scim_endpoint"].(map[string]any)["token"].(string)
		require.NotEmpty(t, scimToken, "the token is returned once at creation and never again")

		scimReq := func(t *testing.T, method, path, bearer, payload string) (int, []byte) {
			t.Helper()
			var rdr io.Reader
			if payload != "" {
				rdr = strings.NewReader(payload)
			}
			req, err := http.NewRequest(method, baseURL+path, rdr)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/scim+json")
			if bearer != "" {
				req.Header.Set("Authorization", "Bearer "+bearer)
			}
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			b, _ := io.ReadAll(resp.Body)
			return resp.StatusCode, b
		}

		// Fail closed first: an unauthenticated call must never provision.
		status, _ := scimReq(t, http.MethodGet, "/scim/v2/Users", "", "")
		assert.Equal(t, http.StatusUnauthorized, status, "SCIM must refuse an unauthenticated caller")

		// Provision a user the way an IdP does.
		scimEmail := "scim-smoke@test.dev"
		payload := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],` +
			`"userName":"` + scimEmail + `","active":true,` +
			`"emails":[{"value":"` + scimEmail + `","primary":true}]}`
		status, body := scimReq(t, http.MethodPost, "/scim/v2/Users", scimToken, payload)
		require.Equal(t, http.StatusCreated, status, "SCIM create failed: %s", body)

		var createdUser struct {
			ID       string `json:"id"`
			UserName string `json:"userName"`
			Active   bool   `json:"active"`
		}
		require.NoError(t, json.Unmarshal(body, &createdUser))
		require.NotEmpty(t, createdUser.ID)
		assert.Equal(t, scimEmail, createdUser.UserName)
		assert.True(t, createdUser.Active)

		// And read it back through the same surface.
		status, body = scimReq(t, http.MethodGet, "/scim/v2/Users/"+createdUser.ID, scimToken, "")
		require.Equal(t, http.StatusOK, status, "SCIM get failed: %s", body)
		assert.Contains(t, string(body), scimEmail)
	})

	// --- Surface 4: MCP over HTTP ----------------------------------------
	// Runs against the REAL binary and the REAL route table, which is the only
	// place the --mcp-enabled route registration is actually exercised: every
	// other MCP test mounts the handler onto a router it builds itself, so a
	// refactor that dropped the route (or the flag guard in front of it) would
	// leave them all green.
	t.Run("mcp http", func(t *testing.T) {
		metadataURL := baseURL + "/.well-known/oauth-protected-resource/mcp"

		// RFC 9728 §5.1: an unauthenticated call must point the client at the
		// metadata document. This is the entry point of the whole discovery
		// chain — without it a fresh client has no way to learn where to
		// authenticate.
		resp, err := http.Post(baseURL+"/mcp", "application/json", strings.NewReader(mcpInitializeRPC))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("WWW-Authenticate"), `resource_metadata="`+metadataURL+`"`)

		// The metadata document the challenge points at must exist and name the
		// same resource identifier clients will send as `resource`.
		metaResp, err := http.Get(metadataURL)
		require.NoError(t, err)
		defer func() { _ = metaResp.Body.Close() }()
		require.Equal(t, http.StatusOK, metaResp.StatusCode)
		var prm struct {
			Resource             string   `json:"resource"`
			AuthorizationServers []string `json:"authorization_servers"`
		}
		require.NoError(t, json.NewDecoder(metaResp.Body).Decode(&prm))
		assert.Equal(t, baseURL+"/mcp", prm.Resource)
		assert.Equal(t, []string{baseURL}, prm.AuthorizationServers)

		// A login token authenticates GraphQL, REST and gRPC in this same test —
		// it must not authenticate MCP. That is the audience boundary, observed
		// from outside the process.
		req, err := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader(mcpInitializeRPC))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer "+token)
		wrongAud, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { _ = wrongAud.Body.Close() }()
		assert.Equal(t, http.StatusUnauthorized, wrongAud.StatusCode,
			"a token minted for the client, not for <url>/mcp, must be refused")
	})

	// --- Surface 4b: MCP over HTTP with an RFC 8693 delegated token -------
	// An agent acting for the user must reach the tools, and must be answered
	// with ITS OWN authority (perms(agent) ∩ perms(user)) rather than the
	// user's. Run against the real binary because the property spans the token
	// endpoint, the audience check and the FGA subject expansion — three
	// components that are individually tested and could still disagree once
	// wired together.
	t.Run("mcp delegated", func(t *testing.T) {
		resource := baseURL + "/mcp"

		created := gql.mutate(t, `mutation { _create_client(params:{name:"smoke-agent", allowed_scopes:["openid"]}) { client { client_id } client_secret } }`)
		agent := created["_create_client"].(map[string]any)
		agentID := agent["client"].(map[string]any)["client_id"].(string)
		agentSecret := agent["client_secret"].(string)
		require.NotEmpty(t, agentID)

		postForm := func(form url.Values) map[string]any {
			resp, err := http.Post(baseURL+"/oauth/token",
				"application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			raw, _ := io.ReadAll(resp.Body)
			require.Equal(t, http.StatusOK, resp.StatusCode, "token endpoint: %s", raw)
			var out map[string]any
			require.NoError(t, json.Unmarshal(raw, &out))
			return out
		}

		ccForm := url.Values{}
		ccForm.Set("grant_type", "client_credentials")
		ccForm.Set("client_id", agentID)
		ccForm.Set("client_secret", agentSecret)
		actorToken := postForm(ccForm)["access_token"].(string)

		exchange := func(res string) string {
			form := url.Values{}
			form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
			form.Set("client_id", agentID)
			form.Set("client_secret", agentSecret)
			form.Set("subject_token", token)
			form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
			form.Set("actor_token", actorToken)
			form.Set("actor_token_type", "urn:ietf:params:oauth:token-type:access_token")
			form.Set("resource", res)
			return postForm(form)["access_token"].(string)
		}

		delegated := exchange(resource)
		apiBound := exchange(baseURL)

		mcpCall := func(bearer, body string) *http.Response {
			req, err := http.NewRequest(http.MethodPost, resource, strings.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			req.Header.Set("Authorization", "Bearer "+bearer)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			return resp
		}

		// The bijection: each token opens exactly the surface its audience names.
		refused := mcpCall(apiBound, mcpInitializeRPC)
		defer func() { _ = refused.Body.Close() }()
		assert.Equal(t, http.StatusUnauthorized, refused.StatusCode,
			"a delegated token bound to the bare server URL must not open /mcp")

		accepted := mcpCall(delegated, mcpInitializeRPC)
		defer func() { _ = accepted.Body.Close() }()
		require.Equal(t, http.StatusOK, accepted.StatusCode, "delegated token must reach /mcp")

		// The intersection, through the real tool. The user is a viewer of
		// document:readme (seeded above); this agent holds no grant at all, so
		// every answer must be false — including the one the user can see.
		callResp := mcpCall(delegated, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"check_permissions",`+
			`"arguments":{"checks":[{"relation":"viewer","object":"document:readme"}]}}}`)
		defer func() { _ = callResp.Body.Close() }()
		require.Equal(t, http.StatusOK, callResp.StatusCode)

		var toolOut struct {
			Result struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
				IsError bool `json:"isError"`
			} `json:"result"`
		}
		require.NoError(t, json.NewDecoder(callResp.Body).Decode(&toolOut))
		require.False(t, toolOut.Result.IsError)
		require.NotEmpty(t, toolOut.Result.Content)

		var perms struct {
			Results []struct {
				Allowed bool `json:"allowed"`
			} `json:"results"`
		}
		require.NoError(t, json.Unmarshal([]byte(toolOut.Result.Content[0].Text), &perms))
		require.Len(t, perms.Results, 1)
		assert.False(t, perms.Results[0].Allowed,
			"CONFUSED DEPUTY: the agent holds no grant, so it must be denied even "+
				"though the delegating user is a viewer of document:readme")
	})

	// --- Surface 5: MCP (stdio subprocess, deprecated) --------------------
	// The MCP subcommand is a separate process sharing the sqlite store, so
	// stop the server first to avoid two writers on one sqlite file.
	stopServer()

	t.Run("mcp stdio", func(t *testing.T) {
		mcpArgs := []string{"mcp",
			"--database-type=sqlite", "--database-url=" + dbPath,
			"--jwt-type=HS256", "--jwt-secret=" + smokeJWTSecret,
			"--admin-secret=" + smokeAdminSecret,
			"--client-id=" + smokeClientID, "--client-secret=" + smokeClientSecret,
			// Replaces --mcp-authorizer-url, inert as of 2.4.0. This is what the
			// bearer's iss claim is validated against.
			"--url=" + baseURL,
			"--mcp-bearer=" + token,
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
		// Profile returns the flat User object (no wrapper), matching the
		// GraphQL `profile` response.
		var profOut struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(profile), &profOut))
		assert.Equal(t, smokeUserEmail, profOut.Email)
	})
}

// mcpInitializeRPC is the MCP handshake a client sends first.
const mcpInitializeRPC = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{` +
	`"protocolVersion":"2025-06-18","capabilities":{},` +
	`"clientInfo":{"name":"release-smoke","version":"1.0"}}}`

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

// adminREST calls a /v1/admin path with the given method, authenticating via
// the x-authorizer-admin-secret header (the admin surface's header auth). An
// empty adminSecret sends no auth header (for fail-closed assertions). An empty
// body sends no request body (for GET endpoints). Returns the HTTP status.
func adminREST(t *testing.T, baseURL, method, path, adminSecret, body string, out any) int {
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
