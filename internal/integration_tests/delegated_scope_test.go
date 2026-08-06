package integration_tests

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/delegatedscope"
)

// TestDelegatedTokenScopeIsEnforced is the regression test for the blast-radius
// vulnerability: a delegated token authenticated at Authorizer's own API and
// reached EVERY first-party operation with its `scope` claim never consulted.
//
// An agent an operator granted `openid` for a downstream MCP server could read
// the delegating user's profile, mutate the account, and deactivate it. The
// RFC 8693 attenuation that produced that scope was computed, returned to the
// caller, and then ignored.
//
// Driven through the REAL /graphql endpoint, not the GraphQLProvider directly.
// Enforcement lives in gqlgen middleware, so a test that calls the service
// layer bypasses it entirely and would pass no matter what the gate did — the
// first version of this test did exactly that.
func TestDelegatedTokenScopeIsEnforced(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	delegated, _, _ := mintDelegatedViaEndpoint(t, ts, tokenRouter, testAuthorizerHost(ts))

	claims, err := ts.TokenProvider.ParseJWTToken(delegated)
	require.NoError(t, err)
	t.Logf("delegated scope: %v", claims["scope"])

	router := setupTestRouter(ts)
	post := func(t *testing.T, query string) string {
		t.Helper()
		body := `{"query":` + jsonQuote(query) + `}`
		w := sendTestRequest(t, router, "POST", "/graphql", body, map[string]string{
			"Content-Type":     "application/json",
			"Authorization":    "Bearer " + delegated,
			"Origin":           "http://localhost:3000",
			"X-Authorizer-URL": testAuthorizerHost(ts),
		})
		return w.Body.String()
	}

	t.Run("read-only permission queries stay reachable", func(t *testing.T) {
		// What the agent feature exists for, and exactly the tool set the
		// built-in MCP server exposes. Gating these would make it useless.
		out := post(t, `query { check_permissions(params: {checks: [{relation: "can_view", object: "document:x"}]}) { results { allowed } } }`)
		assert.NotContains(t, out, "insufficient_scope",
			"check_permissions requires only openid and must stay reachable")
	})

	t.Run("profile stays readable", func(t *testing.T) {
		out := post(t, `query { profile { id email } }`)
		assert.NotContains(t, out, "insufficient_scope",
			"profile requires only openid and must stay reachable")
	})

	t.Run("update_profile is REFUSED", func(t *testing.T) {
		out := post(t, `mutation { update_profile(params: {given_name: "mutated-by-agent"}) { message } }`)
		assert.Contains(t, out, "insufficient_scope",
			"an agent scoped openid/email/profile must not be able to mutate the account")
	})

	t.Run("deactivate_account is REFUSED", func(t *testing.T) {
		out := post(t, `mutation { deactivate_account { message } }`)
		assert.Contains(t, out, "insufficient_scope",
			"an agent must never be able to deactivate its delegating user's account")
	})

	t.Run("an unlisted operation fails closed", func(t *testing.T) {
		// Nothing cleared webauthn_credentials for delegated callers, so it is
		// denied without anyone having had to remember to deny it.
		out := post(t, `query { webauthn_credentials { id } }`)
		assert.Contains(t, out, "insufficient_scope",
			"operations absent from the table must be unreachable by agents by default")
	})
}

// TestFirstPartyTokensAreNotScopeGated pins the deliberate asymmetry.
//
// A first-party `scope` is caller-supplied and unvalidated (service.Login takes
// params.Scope with no allow-list), so it is a hint, not a boundary. Gating on
// it would break every existing client while granting no security. The gate
// exists only where the ceiling is admin-controlled: delegated tokens.
func TestFirstPartyTokensAreNotScopeGated(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := setupTestRouter(ts)

	token := testAccessToken(t, ts)
	body := `{"query":` + jsonQuote(`mutation { update_profile(params: {given_name: "changed-by-the-user"}) { message } }`) + `}`
	w := sendTestRequest(t, router, "POST", "/graphql", body, map[string]string{
		"Content-Type":     "application/json",
		"Authorization":    "Bearer " + token,
		"Origin":           "http://localhost:3000",
		"X-Authorizer-URL": testAuthorizerHost(ts),
	})
	assert.NotContains(t, w.Body.String(), "insufficient_scope",
		"a user updating their own profile must be unaffected by delegated-scope enforcement")
}

// TestDelegatedScopeTableCoversBothTransports guards the one way the two
// enforcement points could disagree: an operation listed for GraphQL but not
// gRPC (or vice versa) would be reachable on one transport and denied on the
// other — the same class of bug as delegation being inert on GraphQL while
// working on gRPC.
func TestDelegatedScopeTableCoversBothTransports(t *testing.T) {
	pairs := []struct{ graphQL, grpc string }{
		{"check_permissions", "CheckPermissions"},
		{"list_permissions", "ListPermissions"},
		{"profile", "Profile"},
		{"meta", "Meta"},
		{"update_profile", "UpdateProfile"},
		{"deactivate_account", "DeactivateAccount"},
	}
	for _, p := range pairs {
		gqlScope, gqlOK := delegatedscope.RequiredForGraphQL(p.graphQL)
		grpcScope, grpcOK := delegatedscope.RequiredForGRPC("/authorizer.v1.AuthorizerService/" + p.grpc)
		require.True(t, gqlOK, "%s missing from the GraphQL side of the table", p.graphQL)
		require.True(t, grpcOK, "%s missing from the gRPC side of the table", p.grpc)
		assert.Equal(t, gqlScope, grpcScope,
			"%s/%s must require the same scope on both transports", p.graphQL, p.grpc)
	}

	_, ok := delegatedscope.RequiredForGRPC("/authorizer.v1.AuthorizerService/SomethingNobodyCleared")
	assert.False(t, ok, "an unknown method must not resolve to a scope")
}

// jsonQuote escapes a GraphQL document for embedding in a JSON request body.
func jsonQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}
