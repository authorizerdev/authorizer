package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// This file tests agent delegation THROUGH THE REAL ENDPOINTS — /oauth/token to
// mint the token, then the public GraphQL permission API with that token on the
// request, exactly as a caller would.
//
// It exists because an earlier round of tests called CreateDelegatedAccessToken
// directly and asserted on ValidateDelegatedAccessToken in isolation. Both
// passed while the feature was entirely unreachable in production: /oauth/token
// requires `resource` to be an absolute URI and stamps it verbatim as `aud`,
// and the validator required aud to equal an opaque client id — conditions that
// can never both hold. Unit-testing the pieces proved nothing about the system.
//
// Every test here starts at an HTTP endpoint.

// fgaAgentModel declares `agent` as a first-class subject. Declaring the type IS
// the opt-in, exactly as `service_account` works (see fgaServiceAccountModel).
const fgaAgentModel = `model
  schema 1.1
type user
type agent
type document
  relations
    define viewer: [user, agent]
    define can_view: viewer
`

// mintDelegatedViaEndpoint performs a real RFC 8693 exchange against
// /oauth/token and returns (delegatedToken, agentClientID, delegatingUserID).
func mintDelegatedViaEndpoint(t *testing.T, ts *testSetup, router http.Handler, resource string) (string, string, string) {
	t.Helper()

	agentClientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
	subjectToken := testAccessToken(t, ts)
	actor := agentAccessToken(t, ts, router, agentClientID, secret)

	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", accessTokenType)
	form.Set("actor_token", actor)
	form.Set("actor_token_type", accessTokenType)
	form.Set("resource", resource)

	// Use the shared helper: it sets X-Authorizer-URL to the host the minted
	// subject/actor tokens use as their iss, without which the exchange rejects
	// them as invalid.
	rec := postTokenExchange(ts, router, form, agentClientID, secret)

	require.Equal(t, http.StatusOK, rec.Code,
		"token exchange must succeed for resource=%q; body=%s", resource, rec.Body.String())

	var out struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.NotEmpty(t, out.AccessToken)

	claims, err := ts.TokenProvider.ParseJWTToken(subjectToken)
	require.NoError(t, err)
	userID, _ := claims["sub"].(string)
	require.NotEmpty(t, userID)

	return out.AccessToken, agentClientID, userID
}

// presentDelegatedToken puts a delegated token on the shared test request, the
// same way presentMachineToken does for client_credentials callers.
func presentDelegatedToken(ts *testSetup, token string) {
	clearCookies(ts)
	ts.GinContext.Request.Header.Set("Authorization", "Bearer "+token)
}

// TestAgentDelegatedTokenReachesAuthorizerAPI is the reachability acceptance
// test: a token minted by the REAL endpoint must authenticate at Authorizer's
// own API when it names Authorizer as its resource, and must not when it names
// someone else's.
func TestAgentDelegatedTokenReachesAuthorizerAPI(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	t.Run("resource = authorizer's own URL authenticates", func(t *testing.T) {
		delegated, agentID, userID := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))
		presentDelegatedToken(ts, delegated)

		data, err := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(ts.GinContext)
		require.NoError(t, err,
			"a delegated token naming authorizer as its resource must authenticate at authorizer's own API; "+
				"without this the whole agent feature is unreachable")
		assert.Equal(t, userID, data.UserID, "the subject stays the delegating user")
		assert.Equal(t, agentID, data.ActorID, "the immediate actor must survive as the agent")
	})

	t.Run("resource = another server is refused here", func(t *testing.T) {
		delegated, _, _ := mintDelegatedViaEndpoint(t, ts, router, "https://mcp.example.com")
		presentDelegatedToken(ts, delegated)

		_, err := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(ts.GinContext)
		require.Error(t, err,
			"a token bound to a downstream resource server must never authenticate here, "+
				"or the RFC 8707 audience restriction is decorative")
	})
}

// TestAgentIntersectionThroughGraphQL is the Confused Deputy acceptance test,
// driven through the public GraphQL API with a real delegated token.
func TestAgentIntersectionThroughGraphQL(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaAgentModel})
	require.NoError(t, err)

	delegated, agentID, userID := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))

	// The USER can view the document. The AGENT has no grant of its own.
	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
		Tuples: []*model.FgaTupleInput{
			{User: "user:" + userID, Relation: "viewer", Object: "document:secret"},
		},
	})
	require.NoError(t, err)

	check := func(t *testing.T, explicitUser *string) *model.CheckPermissionsResponse {
		t.Helper()
		presentDelegatedToken(ts, delegated)
		res, cErr := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			User:   explicitUser,
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:secret"}},
		})
		require.NoError(t, cErr)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)
		return res
	}

	t.Run("agent WITHOUT its own grant is denied though the user has access", func(t *testing.T) {
		assert.False(t, check(t, nil).Results[0].Allowed,
			"CONFUSED DEPUTY: the agent holds no grant, so it must be denied even though "+
				"the delegating user can view the document")
	})

	t.Run("explicit self user must not drop the agent half", func(t *testing.T) {
		// fga.go honours self-specification for ANY caller, so a delegated agent
		// echoing back its own subject previously bypassed the agent check —
		// a one-parameter defeat of the intersection.
		self := "user:" + userID
		assert.False(t, check(t, &self).Results[0].Allowed,
			"supplying an explicit self `user` must not drop the agent half")
	})

	t.Run("bare-id explicit user must not drop the agent half", func(t *testing.T) {
		bare := userID
		assert.False(t, check(t, &bare).Results[0].Allowed,
			"normalizeFgaSubject expands a bare id to user:<id>; that path must not bypass either")
	})

	t.Run("granting the agent allows the intersection", func(t *testing.T) {
		setAdminCookie(t, ts)
		_, wErr := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "agent:" + agentID, Relation: "viewer", Object: "document:secret"},
			},
		})
		require.NoError(t, wErr)
		assert.True(t, check(t, nil).Results[0].Allowed, "both halves granted must allow")
	})

	t.Run("revoking one agent leaves the delegating user untouched", func(t *testing.T) {
		setAdminCookie(t, ts)
		_, dErr := ts.GraphQLProvider.FgaDeleteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "agent:" + agentID, Relation: "viewer", Object: "document:secret"},
			},
		})
		require.NoError(t, dErr)
		assert.False(t, check(t, nil).Results[0].Allowed, "the revoked agent is denied")

		allowed, cErr := eng.Check(ctx, "user:"+userID, "can_view", "document:secret")
		require.NoError(t, cErr)
		assert.True(t, allowed, "revoking one agent must not affect the delegating user")
	})
}

// TestAgentIntersectionListPermissions pins that ENUMERATION intersects too.
// Without it an agent that cannot act on an object would still see it listed,
// leaking the delegating user's resource names.
func TestAgentIntersectionListPermissions(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaAgentModel})
	require.NoError(t, err)

	delegated, _, userID := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))

	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
		Tuples: []*model.FgaTupleInput{
			{User: "user:" + userID, Relation: "viewer", Object: "document:listed"},
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
	assert.Empty(t, res.Objects,
		"the agent holds no grant, so enumeration must be empty — listing an object the "+
			"agent cannot act on leaks the delegating user's resource names")
}
