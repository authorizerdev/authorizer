//go:build smoke

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// conditionModelDSL grants agents time-limited access: an agent tuple only
// holds while non_expired_grant is true. Users keep unconditional grants.
const conditionModelDSL = `model
  schema 1.1
type user
type agent
type document
  relations
    define viewer: [user, agent with non_expired_grant]
condition non_expired_grant(current_time: timestamp, grant_time: timestamp, grant_duration: duration) {
  current_time < grant_time + grant_duration
}`

// TestReleaseSmokeFgaConditions is the release gate for conditional FGA
// grants: the real binary, a real RFC 8693 delegated token, and the same
// decision on GraphQL, REST, gRPC and MCP — including the rules that stop an
// agent from reviving or self-granting access.
func TestReleaseSmokeFgaConditions(t *testing.T) {
	bin := buildBinary(t)
	dbPath := filepath.Join(t.TempDir(), "conditions.db")
	httpPort, metricsPort, grpcPort := freePort(t), freePort(t), freePort(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)
	startServer(t, bin, []string{
		"--database-type=sqlite", "--database-url=" + dbPath,
		"--jwt-type=HS256", "--jwt-secret=" + smokeJWTSecret,
		"--admin-secret=" + smokeAdminSecret,
		"--client-id=" + smokeClientID, "--client-secret=" + smokeClientSecret,
		fmt.Sprintf("--http-port=%d", httpPort),
		fmt.Sprintf("--metrics-port=%d", metricsPort),
		fmt.Sprintf("--grpc-port=%d", grpcPort),
		"--disable-mfa",
		"--mcp-enabled",
		"--url=" + baseURL,
	}, baseURL)

	gql := newGraphQLClient(t, baseURL)
	gql.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)
	gql.mutate(t, `mutation { _fga_write_model(params:{dsl:"`+strings.ReplaceAll(conditionModelDSL, "\n", `\n`)+`"}) { id } }`)

	signup := gql.mutate(t, `mutation { signup(params:{email:"conditions@test.dev", password:"`+smokeUserPassword+`", confirm_password:"`+smokeUserPassword+`"}) { access_token user { id } } }`)
	userToken := signup["signup"].(map[string]any)["access_token"].(string)
	userID := signup["signup"].(map[string]any)["user"].(map[string]any)["id"].(string)

	created := gql.mutate(t, `mutation { _create_client(params:{name:"conditions-agent", allowed_scopes:["openid"]}) { client { client_id } client_secret } }`)
	agentID := created["_create_client"].(map[string]any)["client"].(map[string]any)["client_id"].(string)
	agentSecret := created["_create_client"].(map[string]any)["client_secret"].(string)

	// Grants: the user can view both documents; the agent's grant on
	// document:fresh is current, its grant on document:stale expired an hour
	// ago. Written over GraphQL as object literals, the way the docs show.
	now := time.Now().UTC()
	grant := func(start time.Time) string {
		return `condition:{name:"non_expired_grant", context:{grant_time:"` + start.Format(time.RFC3339) + `", grant_duration:"10m"}}`
	}
	gql.mutate(t, `mutation { _fga_write_tuples(params:{tuples:[`+
		`{user:"user:`+userID+`", relation:"viewer", object:"document:fresh"},`+
		`{user:"user:`+userID+`", relation:"viewer", object:"document:stale"},`+
		`{user:"agent:`+agentID+`", relation:"viewer", object:"document:fresh", `+grant(now)+`},`+
		`{user:"agent:`+agentID+`", relation:"viewer", object:"document:stale", `+grant(now.Add(-time.Hour))+`}`+
		`]}) { message } }`)

	// Real RFC 8693 delegation: one token for Authorizer's API, one for /mcp.
	postForm := func(form url.Values) map[string]any {
		resp, err := http.Post(baseURL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.StatusCode, "token endpoint: %s", raw)
		var out map[string]any
		require.NoError(t, json.Unmarshal(raw, &out))
		return out
	}
	cc := url.Values{"grant_type": {"client_credentials"}, "client_id": {agentID}, "client_secret": {agentSecret}}
	actorToken := postForm(cc)["access_token"].(string)
	exchange := func(resource string) string {
		return postForm(url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"client_id":          {agentID},
			"client_secret":      {agentSecret},
			"subject_token":      {userToken},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
			"actor_token":        {actorToken},
			"actor_token_type":   {"urn:ietf:params:oauth:token-type:access_token"},
			"resource":           {resource},
		})["access_token"].(string)
	}
	delegated := exchange(baseURL)
	delegatedMCP := exchange(baseURL + "/mcp")

	twoChecks := `{"checks":[{"relation":"viewer","object":"document:fresh"},{"relation":"viewer","object":"document:stale"}]}`
	type results struct {
		Results []struct {
			Allowed bool `json:"allowed"`
		} `json:"results"`
	}
	type restErr struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	t.Run("graphql: unexpired grant allows, expired denies", func(t *testing.T) {
		// Cookie-less on purpose: the shared client's jar holds the user's
		// session from signup, and a session cookie takes precedence over the
		// bearer, which would answer as the user instead of the agent.
		data, errs := conditionsGQL(t, baseURL, delegated, `query { check_permissions(params:{checks:[{relation:"viewer", object:"document:fresh"},{relation:"viewer", object:"document:stale"}]}) { results { allowed } } }`)
		require.Empty(t, errs)
		r := data["check_permissions"].(map[string]any)["results"].([]any)
		require.Len(t, r, 2)
		assert.True(t, r[0].(map[string]any)["allowed"].(bool))
		assert.False(t, r[1].(map[string]any)["allowed"].(bool))
	})

	t.Run("graphql: delegated token cannot send contextual tuples", func(t *testing.T) {
		_, errs := conditionsGQL(t, baseURL, delegated, `query { check_permissions(params:{checks:[{relation:"viewer", object:"document:stale", contextual_tuples:[{user:"agent:`+agentID+`", relation:"viewer", object:"document:stale"}]}]}) { results { allowed } } }`)
		require.NotEmpty(t, errs)
		assert.Contains(t, fmt.Sprint(errs), "may not supply contextual tuples")
	})

	t.Run("rest: same decision, and list_permissions honours expiry", func(t *testing.T) {
		var out results
		require.Equal(t, http.StatusOK, restJSON(t, baseURL, "/v1/check_permissions", delegated, twoChecks, &out))
		require.Len(t, out.Results, 2)
		assert.True(t, out.Results[0].Allowed)
		assert.False(t, out.Results[1].Allowed)

		var list struct {
			Objects []string `json:"objects"`
		}
		require.Equal(t, http.StatusOK, restJSON(t, baseURL, "/v1/list_permissions", delegated, `{"relation":"viewer","object_type":"document"}`, &list))
		assert.Equal(t, []string{"document:fresh"}, list.Objects)
	})

	t.Run("rest: agents and users cannot send context; agents cannot send contextual tuples", func(t *testing.T) {
		forged := `{"checks":[{"relation":"viewer","object":"document:stale"}],"context":{"current_time":"2000-01-01T00:00:00Z"}}`
		var e restErr
		assert.Equal(t, http.StatusForbidden, restJSON(t, baseURL, "/v1/check_permissions", delegated, forged, &e))
		assert.Equal(t, http.StatusForbidden, restJSON(t, baseURL, "/v1/check_permissions", userToken, forged, &e))
		selfGrant := `{"checks":[{"relation":"viewer","object":"document:stale","contextual_tuples":[{"user":"agent:` + agentID + `","relation":"viewer","object":"document:stale"}]}]}`
		assert.Equal(t, http.StatusForbidden, restJSON(t, baseURL, "/v1/check_permissions", delegated, selfGrant, &e))
	})

	t.Run("rest admin: write rules and read-back", func(t *testing.T) {
		var e restErr
		missing := `{"tuples":[{"user":"agent:x","relation":"viewer","object":"document:x","condition":{"name":"non_expired_grant","context":{"grant_time":"2026-01-01T00:00:00Z"}}}]}`
		assert.Equal(t, http.StatusBadRequest, adminREST(t, baseURL, http.MethodPost, "/v1/admin/fga/tuples", smokeAdminSecret, missing, &e))
		assert.Contains(t, e.Message, "missing parameters grant_duration")

		frozen := `{"tuples":[{"user":"agent:x","relation":"viewer","object":"document:x","condition":{"name":"non_expired_grant","context":{"grant_time":"2026-01-01T00:00:00Z","grant_duration":"1m","current_time":"2026-01-01T00:00:00Z"}}}]}`
		assert.Equal(t, http.StatusBadRequest, adminREST(t, baseURL, http.MethodPost, "/v1/admin/fga/tuples", smokeAdminSecret, frozen, &e))

		var read struct {
			Tuples []struct {
				User      string `json:"user"`
				Condition *struct {
					Name    string         `json:"name"`
					Context map[string]any `json:"context"`
				} `json:"condition"`
			} `json:"tuples"`
		}
		require.Equal(t, http.StatusOK, adminREST(t, baseURL, http.MethodPost, "/v1/admin/fga/tuples/read", smokeAdminSecret, `{"object":"document:fresh"}`, &read))
		var agentCond, userCond bool
		for _, tp := range read.Tuples {
			switch tp.User {
			case "agent:" + agentID:
				agentCond = tp.Condition != nil && tp.Condition.Name == "non_expired_grant" && tp.Condition.Context["grant_duration"] == "10m"
			case "user:" + userID:
				userCond = tp.Condition == nil
			}
		}
		assert.True(t, agentCond, "the agent tuple reads back with its condition")
		assert.True(t, userCond, "the plain tuple reads back with no condition")
	})

	t.Run("grpc: same decision; context and contextual tuples refused for the agent", func(t *testing.T) {
		conn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		client := authorizerv1.NewAuthorizerServiceClient(conn)
		ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+delegated, "x-authorizer-url", baseURL)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		res, err := client.CheckPermissions(ctx, &authorizerv1.CheckPermissionsRequest{Checks: []*authorizerv1.PermissionCheckInput{
			{Relation: "viewer", Object: "document:fresh"},
			{Relation: "viewer", Object: "document:stale"},
		}})
		require.NoError(t, err)
		require.Len(t, res.Results, 2)
		assert.True(t, res.Results[0].Allowed)
		assert.False(t, res.Results[1].Allowed)

		forged, err := structpb.NewStruct(map[string]any{"current_time": "2000-01-01T00:00:00Z"})
		require.NoError(t, err)
		_, err = client.CheckPermissions(ctx, &authorizerv1.CheckPermissionsRequest{
			Checks:  []*authorizerv1.PermissionCheckInput{{Relation: "viewer", Object: "document:stale"}},
			Context: forged,
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))

		_, err = client.CheckPermissions(ctx, &authorizerv1.CheckPermissionsRequest{
			Checks: []*authorizerv1.PermissionCheckInput{{Relation: "viewer", Object: "document:stale",
				ContextualTuples: []*authorizerv1.FgaTupleInput{{User: "agent:" + agentID, Relation: "viewer", Object: "document:stale"}}}},
		})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("mcp: same decision through the tool; self-grant refused", func(t *testing.T) {
		callTool := func(args string) (text string, isError bool) {
			req, err := http.NewRequest(http.MethodPost, baseURL+"/mcp", strings.NewReader(
				`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"check_permissions","arguments":`+args+`}}`))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			req.Header.Set("Authorization", "Bearer "+delegatedMCP)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			require.Equal(t, http.StatusOK, resp.StatusCode)
			var out struct {
				Result struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
			require.NotEmpty(t, out.Result.Content)
			return out.Result.Content[0].Text, out.Result.IsError
		}

		text, isErr := callTool(twoChecks)
		require.False(t, isErr, text)
		var r results
		require.NoError(t, json.Unmarshal([]byte(text), &r))
		require.Len(t, r.Results, 2)
		assert.True(t, r.Results[0].Allowed)
		assert.False(t, r.Results[1].Allowed)

		text, isErr = callTool(`{"checks":[{"relation":"viewer","object":"document:stale","contextual_tuples":[{"user":"agent:` + agentID + `","relation":"viewer","object":"document:stale"}]}]}`)
		assert.True(t, isErr)
		assert.Contains(t, text, "may not supply contextual tuples")

		text, isErr = callTool(`{"checks":[{"relation":"viewer","object":"document:stale"}],"context":{"current_time":"2000-01-01T00:00:00Z"}}`)
		assert.True(t, isErr)
		assert.Contains(t, text, "may not supply condition context")
	})
}

// conditionsGQL runs a GraphQL query with only a bearer token (no cookie jar) and
// returns its data and errors. The shared client fails the test on any error,
// which the rejection cases need to inspect instead.
func conditionsGQL(t *testing.T, baseURL, bearer, query string) (map[string]any, []any) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/graphql", strings.NewReader(string(body)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		Data   map[string]any `json:"data"`
		Errors []any          `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out.Data, out.Errors
}
