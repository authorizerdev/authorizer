package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TestAgentEnumerationWhenUserIsTheRestrictiveSide closes a hole in
// TestAgentIntersectionListPermissions.
//
// That test grants the agent NOTHING, so the agent's own enumeration is empty
// and the intersection is empty for a trivial reason. Because the agent is
// subjects[0], returning the agent's set unintersected produces the SAME empty
// answer — so the assertion passes even if the fold is not performing an
// intersection at all. A mutation that replaced intersectObjects with `return a`
// survived the entire suite.
//
// Here the AGENT is the broader side and the USER is the restrictive one: the
// agent is granted two documents, the delegating user only one. Anything other
// than a real intersection hands the agent an object its user cannot reach,
// which is the Confused Deputy expressed through enumeration.
func TestAgentEnumerationWhenUserIsTheRestrictiveSide(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaAgentModel})
	require.NoError(t, err)

	delegated, agentID, userID := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))

	const shared = "document:shared"
	const agentOnly = "document:agent-only"

	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
		Tuples: []*model.FgaTupleInput{
			// The user can reach only the shared document.
			{User: "user:" + userID, Relation: "viewer", Object: shared},
			// The agent can reach both — it is the BROADER side here.
			{User: "agent:" + agentID, Relation: "viewer", Object: shared},
			{User: "agent:" + agentID, Relation: "viewer", Object: agentOnly},
		},
	})
	require.NoError(t, err)

	presentDelegatedToken(ts, delegated)
	res, lErr := ts.GraphQLProvider.ListPermissions(ctx, &model.ListPermissionsInput{
		ObjectType: refs.NewStringRef("document"),
		Relation:   refs.NewStringRef("can_view"),
	})
	require.NoError(t, lErr)
	require.NotNil(t, res)

	assert.Contains(t, res.Objects, shared,
		"both halves grant the shared document, so it must be enumerated")
	assert.NotContains(t, res.Objects, agentOnly,
		"the agent holds this grant but its delegating user does NOT — enumerating it "+
			"hands the agent an object beyond the user's reach, which is exactly what the "+
			"intersection exists to prevent")
}

// TestDelegatedDenialIsAttributedToTheUser closes the other half of the
// denial-attribution metric.
//
// The suite covered denied_by_agent (agent lacks the grant) and not_enforced,
// but never denied_by_user — the case where the agent HAS its grant and the
// delegating user does not. That is the Confused Deputy actually being stopped,
// and it is the outcome an operator must NOT respond to by widening the agent.
// A mutation that recorded denied_by_agent for both branches survived the suite,
// which would have told operators to grant the agent a tuple that cannot help.
func TestDelegatedDenialIsAttributedToTheUser(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaAgentModel})
	require.NoError(t, err)

	delegated, agentID, _ := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))

	const obj = "document:agent-has-user-does-not"
	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
		// ONLY the agent is granted. The delegating user is not.
		Tuples: []*model.FgaTupleInput{{User: "agent:" + agentID, Relation: "viewer", Object: obj}},
	})
	require.NoError(t, err)

	byUser := func() float64 {
		return testutil.ToFloat64(metrics.FgaDelegatedChecksTotal.WithLabelValues(
			metrics.FgaOpCheckPermissions, metrics.FgaDelegatedDeniedByUser))
	}
	byAgent := func() float64 {
		return testutil.ToFloat64(metrics.FgaDelegatedChecksTotal.WithLabelValues(
			metrics.FgaOpCheckPermissions, metrics.FgaDelegatedDeniedByAgent))
	}
	userBefore, agentBefore := byUser(), byAgent()

	presentDelegatedToken(ts, delegated)
	res, cErr := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
		Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: obj}},
	})
	require.NoError(t, cErr)
	require.Len(t, res.Results, 1)
	require.False(t, res.Results[0].Allowed, "the user lacks the grant, so the intersection denies")

	assert.Equal(t, userBefore+1, byUser(),
		"the USER is the side that refused; recording this as denied_by_agent would tell an "+
			"operator to grant the agent a tuple, which cannot fix it and widens the agent for nothing")
	assert.Equal(t, agentBefore, byAgent(), "the agent had its grant, so it did not deny")
}

// TestChainedDelegationKeepsItsSessionBinding covers MULTI-HOP delegation into
// Authorizer's own API, which nothing else exercises.
//
// A delegated token carries `sid` (the originating session) and NO `nonce`. A
// second hop therefore cannot rebuild the binding from nonce + login_method the
// way a first hop does — it must propagate the incoming `sid` verbatim. Drop
// that and hop 2 mints a token with no `sid`, which fails the session check and
// cannot authenticate here at all: multi-hop agents break, and the failure
// surfaces as a bare "unauthorized" that reads like a permissions problem.
//
// It matters in the other direction too: a hop-2 token that kept working
// WITHOUT a binding would outlive the logout that ended the session it grew
// from.
func TestChainedDelegationKeepsItsSessionBinding(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	host := testAuthorizerHost(ts)

	// Hop 1: user -> agent A, bound to Authorizer itself.
	firstHop, _, userID := mintDelegatedViaEndpoint(t, ts, router, host)

	// Hop 2: agent B re-exchanges agent A's delegated token.
	agentBID, agentBSecret := newDelegationAgent(t, ts, "openid,profile,email")
	agentBActor := agentAccessToken(t, ts, router, agentBID, agentBSecret)

	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", firstHop)
	form.Set("subject_token_type", accessTokenType)
	form.Set("actor_token", agentBActor)
	form.Set("actor_token_type", accessTokenType)
	form.Set("resource", host)
	rec := postTokenExchange(ts, router, form, agentBID, agentBSecret)
	require.Equal(t, http.StatusOK, rec.Code, "the second hop must be permitted; body=%s", rec.Body.String())

	var out struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.NotEmpty(t, out.AccessToken)

	claims, err := ts.TokenProvider.ParseJWTToken(out.AccessToken)
	require.NoError(t, err)
	require.Equal(t, userID, claims["sub"], "the subject stays the original user across hops")
	sid, _ := claims["sid"].(string)
	require.NotEmpty(t, sid,
		"the second hop must carry the originating session forward — a delegated subject_token "+
			"has no nonce to rebuild it from, so dropping the incoming sid leaves the chain unable "+
			"to authenticate here at all")

	// And it must work end to end, not merely carry the claim.
	presentDelegatedToken(ts, out.AccessToken)
	data, aErr := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(ts.GinContext)
	require.NoError(t, aErr, "a two-hop delegated token naming Authorizer must authenticate here")
	assert.Equal(t, userID, data.UserID)
	assert.Equal(t, agentBID, data.ActorID, "the IMMEDIATE actor is the second-hop agent")
}
