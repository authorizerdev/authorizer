package integration_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
)

// This file covers RFC 8693 delegated tokens at the MCP surface: an agent
// holding "agent X acting for user Y" authority calling Authorizer's own MCP
// tools to ask about that authority.
//
// Every test starts at an HTTP endpoint and mints through the REAL
// /oauth/token. That is deliberate and load-bearing: an earlier round of
// delegation tests called CreateDelegatedAccessToken directly and asserted on
// the validator in isolation, and both passed while the feature was entirely
// unreachable in production (see agent_intersection_e2e_test.go). Testing the
// pieces proves nothing about the system.

// mcpTestSetup pins the canonical URL to the running test server and returns
// the MCP resource identifier plus a router serving BOTH /oauth/token (to mint)
// and /mcp (to spend).
//
// The URL pinning is not incidental. MCP requires --url, and with it set
// parsers.GetHost returns the canonical value for every request regardless of
// headers — so a token's `iss`, the delegated validator's issuer check and the
// audience comparison all resolve to one value. Pinning it to the real listener
// address (rather than a fictional host) is what lets tokens minted through the
// real endpoint validate here: the mint helpers stamp testAuthorizerHost(ts) as
// the issuer, and a mismatch would reject every token in this file for a reason
// unrelated to the rule under test.
func mcpTestSetup(t *testing.T, ts *testSetup, cfg *config.Config) (string, http.Handler, http.Handler) {
	t.Helper()

	cfg.MCPEnabled = true
	cfg.AuthorizerURL = testAuthorizerHost(ts)
	parsers.SetTrustedURL(cfg.AuthorizerURL)
	t.Cleanup(func() { parsers.SetTrustedURL("") })

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	// Built after AuthorizerURL is set: mcpRouter captures cfg.MCPResource() at
	// wiring time, exactly as internal/server does.
	return cfg.MCPResource(), tokenRouter, mcpRouter(t, ts, cfg)
}

// mcpToolCall invokes an MCP tool over the real Streamable HTTP transport and
// returns the tool's text content together with its isError flag.
func mcpToolCall(t *testing.T, router http.Handler, bearer, tool string, args map[string]interface{}) (string, bool) {
	t.Helper()

	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)
	body := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":%q,"arguments":%s}}`,
		tool, argsJSON)

	w := mcpPost(t, router, bearer, body)
	require.Equal(t, http.StatusOK, w.Code, "tools/call body: %s", w.Body.String())

	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), "raw: %s", w.Body.String())
	require.NotEmpty(t, resp.Result.Content, "a tool result must carry content; raw: %s", w.Body.String())
	return resp.Result.Content[0].Text, resp.Result.IsError
}

// TestMCPDelegatedTokenReachesMCPSurface is the reachability acceptance test.
//
// It is the one that would have caught the feature being dead: /oauth/token
// requires `resource` to be an absolute URI and stamps it verbatim as `aud`, so
// a token bound to "<url>/mcp" is producible — and before this change it was
// accepted nowhere at all, refused at /mcp by the stateful session check and at
// /graphql by the audience check.
func TestMCPDelegatedTokenReachesMCPSurface(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	resource, tokenRouter, router := mcpTestSetup(t, ts, cfg)

	delegated, _, _ := mintDelegatedViaEndpoint(t, ts, tokenRouter, resource)

	t.Run("handshake succeeds", func(t *testing.T) {
		w := mcpPost(t, router, delegated, initializeRPC)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})

	t.Run("the tool surface is reachable", func(t *testing.T) {
		w := mcpPost(t, router, delegated, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var resp struct {
			Result struct {
				Tools []struct {
					Name string `json:"name"`
				} `json:"tools"`
			} `json:"result"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

		names := make([]string, 0, len(resp.Result.Tools))
		for _, tool := range resp.Result.Tools {
			names = append(names, tool.Name)
		}
		assert.Contains(t, names, "check_permissions",
			"the permission tools are the entire reason an agent presents a delegated token here")
	})
}

// TestMCPDelegatedTokenAudienceBijection pins the invariant the widening had to
// preserve:
//
//	f(aud) = surface, and it is a BIJECTION.
//
// Every token is valid at exactly ONE surface — the one its audience names.
// Both directions are asserted because a mistake in either is a real
// vulnerability, and because the obvious implementation of this feature breaks
// the first one silently:
//
//   - ValidateDelegatedAccessToken has a single caller,
//     GetUserIDFromSessionOrAccessToken, which is the default rule behind
//     /graphql, /v1/* and gRPC. Relaxing its audience check in place — rather
//     than adding ValidateDelegatedAccessTokenForResource — would have made
//     every MCP-bound delegated token a full first-party API credential.
//   - Matching "hostname OR resource" instead of exactly one would break the
//     bijection from the other side, letting a token minted for the first-party
//     API authenticate at /mcp.
func TestMCPDelegatedTokenAudienceBijection(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	resource, tokenRouter, router := mcpTestSetup(t, ts, cfg)
	firstParty := cfg.AuthorizerURL

	mcpBound, _, _ := mintDelegatedViaEndpoint(t, ts, tokenRouter, resource)
	apiBound, _, _ := mintDelegatedViaEndpoint(t, ts, tokenRouter, firstParty)

	t.Run("mcp-bound token is ACCEPTED at /mcp", func(t *testing.T) {
		w := mcpPost(t, router, mcpBound, initializeRPC)
		assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})

	t.Run("mcp-bound token is REJECTED at the first-party API", func(t *testing.T) {
		// The default rule, i.e. what /graphql, /v1/* and gRPC all use.
		_, err := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(
			bearerGinContext(t, ts, mcpBound))
		require.Error(t, err,
			"an MCP token may be handed to a semi-trusted agent; it must never "+
				"become a full GraphQL/REST/gRPC credential")
	})

	t.Run("api-bound token is REJECTED at /mcp", func(t *testing.T) {
		w := mcpPost(t, router, apiBound, initializeRPC)
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"the audience match must be EXACT: a delegated token for the "+
				"first-party API must not also open the MCP surface")
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), `error="invalid_token"`)
	})

	t.Run("api-bound token is still ACCEPTED at the first-party API", func(t *testing.T) {
		// Regression guard: the refactor that added the resource-parameterized
		// entry point must not have narrowed the existing one.
		data, err := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(
			bearerGinContext(t, ts, apiBound))
		require.NoError(t, err)
		require.NotNil(t, data)
		assert.NotEmpty(t, data.ActorID, "the delegation actor must survive validation")
	})

	t.Run("an ordinary login token is still rejected at /mcp", func(t *testing.T) {
		// The delegated fallback is gated on the `act` claim. A non-delegated
		// token must take the unchanged stateful rejection, not the new path.
		w := mcpPost(t, router, testAccessToken(t, ts), initializeRPC)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestMCPDelegatedIntersectionThroughMCP is the security test that actually
// matters for this feature.
//
// The whole point of letting an agent reach /mcp is that the answer it gets is
// its OWN authority — perms(agent) ∩ perms(user) — not the delegating user's.
// If the intersection were dropped on this transport, an agent would ask
// "can I read payroll?", be told yes because the USER can, and the delegation
// model would be decorative exactly where a model is in the loop.
//
// The equivalent GraphQL assertions live in TestAgentIntersectionThroughGraphQL.
// This is the same property observed through the MCP tool surface, because the
// two reach resolveFgaCaller by different routes: GraphQL falls back to the
// request token, while MCP populates authctx.Principal from the interceptor.
func TestMCPDelegatedIntersectionThroughMCP(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)
	resource, tokenRouter, router := mcpTestSetup(t, ts, cfg)

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaAgentModel})
	require.NoError(t, err)

	delegated, agentID, userID := mintDelegatedViaEndpoint(t, ts, tokenRouter, resource)

	// The USER can view the document. The AGENT has no grant of its own.
	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
		Tuples: []*model.FgaTupleInput{
			{User: "user:" + userID, Relation: "viewer", Object: "document:secret"},
		},
	})
	require.NoError(t, err)

	checkSecret := func(t *testing.T, args map[string]interface{}) bool {
		t.Helper()
		text, isErr := mcpToolCall(t, router, delegated, "check_permissions", args)
		require.False(t, isErr, "check_permissions returned an error result: %s", text)

		var out struct {
			Results []struct {
				Allowed bool `json:"allowed"`
			} `json:"results"`
		}
		require.NoError(t, json.Unmarshal([]byte(text), &out), "raw: %s", text)
		require.Len(t, out.Results, 1)
		return out.Results[0].Allowed
	}

	selfCheck := map[string]interface{}{
		"checks": []map[string]string{{"relation": "can_view", "object": "document:secret"}},
	}

	t.Run("agent WITHOUT its own grant is denied though the user has access", func(t *testing.T) {
		assert.False(t, checkSecret(t, selfCheck),
			"CONFUSED DEPUTY: the agent holds no grant, so it must be denied over MCP "+
				"even though the delegating user can view the document")
	})

	t.Run("explicit self user must not drop the agent half", func(t *testing.T) {
		assert.False(t, checkSecret(t, map[string]interface{}{
			"checks": selfCheck["checks"],
			"user":   "user:" + userID,
		}), "echoing back your own subject must not shed the agent constraint")
	})

	t.Run("naming another subject is refused outright", func(t *testing.T) {
		text, isErr := mcpToolCall(t, router, delegated, "check_permissions", map[string]interface{}{
			"checks": selfCheck["checks"],
			"user":   "user:someone-else",
		})
		assert.True(t, isErr,
			"a delegated caller may never widen its subject; got: %s", text)
	})

	t.Run("granting the agent allows the intersection", func(t *testing.T) {
		setAdminCookie(t, ts)
		_, wErr := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "agent:" + agentID, Relation: "viewer", Object: "document:secret"},
			},
		})
		require.NoError(t, wErr)
		assert.True(t, checkSecret(t, selfCheck),
			"both halves granted must allow — otherwise the tool is useless, not merely safe")
	})

	t.Run("list_permissions intersects too", func(t *testing.T) {
		text, isErr := mcpToolCall(t, router, delegated, "list_permissions", map[string]interface{}{
			"relation":    "can_view",
			"object_type": "document",
		})
		require.False(t, isErr, "list_permissions returned an error result: %s", text)
		assert.Contains(t, text, "document:secret",
			"the agent now holds the grant, so enumeration must include it")
	})
}
