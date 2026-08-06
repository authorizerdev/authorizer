package integration_tests

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Attacks the delegated scope gate added in gqlDelegatedScopeMiddleware.
// Every subtest is an attempt to reach update_profile / deactivate_account
// with a token scoped openid/email/profile.
func TestDelegatedScopeGateAdversarial(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	delegated, _, _ := mintDelegatedViaEndpoint(t, ts, tokenRouter, testAuthorizerHost(ts))

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

	t.Run("ATTACK: alias the field name", func(t *testing.T) {
		out := post(t, `mutation { renamed: update_profile(params: {given_name: "via-alias"}) { message } }`)
		require.Contains(t, out, "insufficient_scope",
			"an alias must not bypass the gate — fc.Field.Name must be the schema name, not the alias")
	})

	t.Run("ATTACK: hide the denied field behind an allowed one", func(t *testing.T) {
		out := post(t, `mutation { update_profile(params: {given_name: "hidden"}) { message } }`)
		require.Contains(t, out, "insufficient_scope")
	})

	t.Run("ATTACK: two root fields, one allowed one denied", func(t *testing.T) {
		out := post(t, `query { profile { id } webauthn_credentials { id } }`)
		require.Contains(t, out, "insufficient_scope",
			"a denied root field must still be denied when batched with an allowed one")
	})

	t.Run("ATTACK: named operation", func(t *testing.T) {
		out := post(t, `mutation Evil { update_profile(params: {given_name: "named"}) { message } }`)
		require.Contains(t, out, "insufficient_scope")
	})

	t.Run("ATTACK: fragment spread on the root", func(t *testing.T) {
		out := post(t, `mutation { ...F } fragment F on Mutation { update_profile(params: {given_name: "frag"}) { message } }`)
		require.Contains(t, out, "insufficient_scope",
			"a fragment spread must not route around a root-field check")
	})

	t.Run("ATTACK: __typename alongside a denied field", func(t *testing.T) {
		out := post(t, `mutation { __typename update_profile(params: {given_name: "tn"}) { message } }`)
		require.Contains(t, out, "insufficient_scope")
	})
}
