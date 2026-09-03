package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// newDelegationAgentFull is newDelegationAgent, but returns the whole client so
// a test can assert on the SURROGATE id (schemas.Client.ID) that a machine
// token carries as `sub` — distinct from the public client_id that appears in
// the `act` chain.
func newDelegationAgentFull(t *testing.T, ts *testSetup, ceiling string) (*schemas.Client, string) {
	t.Helper()
	secret := "agent-secret-" + uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	require.NoError(t, err)
	agent, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
		Name:          "agent-" + uuid.New().String(),
		Kind:          constants.ClientKindServiceAccount,
		ClientSecret:  string(hash),
		AllowedScopes: ceiling,
		IsActive:      true,
	})
	require.NoError(t, err)
	return agent, secret
}

// exchangeTokens performs an RFC 8693 exchange and returns the response recorder.
func exchangeTokens(t *testing.T, ts *testSetup, router http.Handler, subjectToken, actorToken, agentClientID, agentSecret, resource string) (int, string) {
	t.Helper()
	form := url.Values{}
	form.Set("grant_type", tokenExchangeGrant)
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", accessTokenType)
	form.Set("actor_token", actorToken)
	form.Set("actor_token_type", accessTokenType)
	form.Set("requested_token_type", accessTokenType)
	form.Set("resource", resource)
	w := postTokenExchange(ts, router, form, agentClientID, agentSecret)
	if w.Code != http.StatusOK {
		return w.Code, ""
	}
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	tok, _ := resp["access_token"].(string)
	return w.Code, tok
}

// TestDelegatedTokenKeepsMachineIdentity is the regression test for
// GHSA-vq29-8q3c-3hrm.
//
// A delegated token minted from a SERVICE-ACCOUNT subject dropped the
// `login_method` claim. service/fga.go classifies a caller with no
// login_method as "user:<sub>", so an autonomous machine identity was
// re-classified as an interactive user — flipping OpenFGA decisions from deny
// to allow and slipping past every login_method-keyed guard.
//
// The report framed this as SELF-delegation (subject == actor) and proposed
// rejecting that shape. That is a symptom patch: the laundering is a property
// of CreateDelegatedAccessToken omitting the claim, so it applies identically
// to the multi-hop agent-to-agent chain the design explicitly supports and
// which the proposed check does NOT cover. Both shapes are asserted here.
func TestDelegatedTokenKeepsMachineIdentity(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	resource := "https://api.example.com/v1"

	t.Run("self delegation: subject and actor are the same service account", func(t *testing.T) {
		agent, secret := newDelegationAgentFull(t, ts, "openid,email")
		machine := agentAccessToken(t, ts, router, agent.ClientID, secret)

		// Sanity: the machine token identifies itself as a service account.
		mc := decodeJWTPayload(t, machine)
		require.Equal(t, constants.AuthRecipeMethodServiceAccount, mc["login_method"])
		require.Equal(t, agent.ID, mc["sub"], "machine token sub is the surrogate id")

		code, delegated := exchangeTokens(t, ts, router, machine, machine, agent.ClientID, secret, resource)
		require.Equal(t, http.StatusOK, code, "self-exchange currently succeeds")

		dc := decodeJWTPayload(t, delegated)
		assert.Equal(t, agent.ID, dc["sub"], "sub is still the service account")
		// THE BUG: without login_method, fga.go resolves this to user:<sub>.
		assert.Equal(t, constants.AuthRecipeMethodServiceAccount, dc["login_method"],
			"a delegated token whose SUBJECT is a service account must keep the machine identity")
	})

	t.Run("multi-hop: agent A delegates agent B's machine identity", func(t *testing.T) {
		// The shape the reporter's subject!=actor check would NOT catch.
		agentA, secretA := newDelegationAgentFull(t, ts, "openid,email")
		agentB, secretB := newDelegationAgentFull(t, ts, "openid,email")

		subjectMachine := agentAccessToken(t, ts, router, agentB.ClientID, secretB)
		actorMachine := agentAccessToken(t, ts, router, agentA.ClientID, secretA)

		code, delegated := exchangeTokens(t, ts, router, subjectMachine, actorMachine, agentA.ClientID, secretA, resource)
		require.Equal(t, http.StatusOK, code, "multi-hop agent chain is a supported shape")

		dc := decodeJWTPayload(t, delegated)
		assert.Equal(t, agentB.ID, dc["sub"], "subject is agent B")
		assert.Equal(t, constants.AuthRecipeMethodServiceAccount, dc["login_method"],
			"agent B's machine identity must survive delegation by agent A")

		act, ok := dc["act"].(map[string]interface{})
		require.True(t, ok, "delegation must record the actor")
		assert.Equal(t, agentA.ClientID, act["sub"], "immediate actor is agent A")
	})

	t.Run("user subject is unchanged: no login_method stamped", func(t *testing.T) {
		// Backward-compatibility guard. A USER-subject delegation must keep
		// resolving to user:<sub>, i.e. carry no login_method, exactly as
		// before. Stamping it here would break every existing delegation.
		agent, secret := newDelegationAgentFull(t, ts, "openid,email,profile")
		actor := agentAccessToken(t, ts, router, agent.ClientID, secret)
		userToken := testAccessToken(t, ts)

		code, delegated := exchangeTokens(t, ts, router, userToken, actor, agent.ClientID, secret, resource)
		require.Equal(t, http.StatusOK, code)

		dc := decodeJWTPayload(t, delegated)
		_, hasLoginMethod := dc["login_method"]
		assert.False(t, hasLoginMethod,
			"a USER-subject delegation must carry no login_method, so it still resolves to user:<sub>")
	})
}

// TestDelegatedMachineTokenRevocationWorks disproves the report's third claim,
// that the machine-derived `sid` ("service_account:<id>|<nonce>") addresses "a
// coordinate machine tokens never occupy", so revocation silently misfires.
//
// It does occupy it: the client_credentials handler registers the machine token
// at session key "service_account:<sa.ID>" under "access_token_<nonce>"
// (internal/http_handlers/token.go), which is exactly what
// delegationSessionIsLive looks up. Deleting that session must kill the
// delegated token at Authorizer's own API.
func TestDelegatedMachineTokenRevocationWorks(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	agent, secret := newDelegationAgentFull(t, ts, "openid,email")
	machine := agentAccessToken(t, ts, router, agent.ClientID, secret)
	mc := decodeJWTPayload(t, machine)
	nonce, _ := mc["nonce"].(string)
	require.NotEmpty(t, nonce, "machine token must carry a nonce")

	sessionKey := constants.AuthRecipeMethodServiceAccount + ":" + agent.ID
	// The session the delegation will bind to must exist right now.
	_, err := ts.MemoryStoreProvider.GetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+nonce)
	require.NoError(t, err, "machine token IS registered at the coordinate the report calls unoccupied")

	code, delegated := exchangeTokens(t, ts, router, machine, machine,
		agent.ClientID, secret, testAuthorizerHost(ts))
	require.Equal(t, http.StatusOK, code)

	dc := decodeJWTPayload(t, delegated)
	assert.Equal(t, sessionKey+"|"+nonce, dc["sid"], "sid addresses the machine token's own session")

	// Revoke by deleting that session, then the delegated token must stop
	// authenticating at Authorizer's own API.
	gc := &gin.Context{}
	gc.Request, _ = http.NewRequest(http.MethodGet, testAuthorizerHost(ts), nil)
	gc.Request.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))

	_, beforeErr := ts.TokenProvider.ValidateDelegatedAccessToken(gc, delegated)
	require.NoError(t, beforeErr, "delegated token should validate while the session is live")

	require.NoError(t, ts.MemoryStoreProvider.DeleteUserSession(sessionKey, nonce))

	_, afterErr := ts.TokenProvider.ValidateDelegatedAccessToken(gc, delegated)
	assert.Error(t, afterErr, "revocation must take the delegated token down with the session")
}

// fgaMachineDelegationModel adds `service_account` alongside `user`/`agent` so
// a machine subject can actually be expressed. fgaAgentModel omits it.
const fgaMachineDelegationModel = `model
  schema 1.1
type user
type agent
type service_account
type document
  relations
    define viewer: [user, agent, service_account]
    define can_view: viewer
`

// TestDelegatedMachineIdentityDoesNotFlipFgaDecision is the end-to-end
// acceptance test for GHSA-vq29-8q3c-3hrm: the deny -> allow flip itself,
// driven through the public GraphQL permission API with a real delegated token
// minted by the real /oauth/token endpoint.
func TestDelegatedMachineIdentityDoesNotFlipFgaDecision(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaMachineDelegationModel})
	require.NoError(t, err)

	agent, secret := newDelegationAgentFull(t, ts, "openid,email")
	machine := agentAccessToken(t, ts, router, agent.ClientID, secret)
	code, delegated := exchangeTokens(t, ts, router, machine, machine,
		agent.ClientID, secret, testAuthorizerHost(ts))
	require.Equal(t, http.StatusOK, code)

	checkAs := func(t *testing.T, tok string) bool {
		t.Helper()
		presentDelegatedToken(ts, tok)
		res, cErr := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:readme"}},
		})
		require.NoError(t, cErr)
		require.NotNil(t, res)
		require.Len(t, res.Results, 1)
		return res.Results[0].Allowed
	}

	t.Run("a user:<sa-id> grant must NOT reach the machine's delegated token", func(t *testing.T) {
		// This is the exploit, reproduced exactly as reported: the delegated
		// token was classified as "user:<sa-row-id>", so a grant written for a
		// user-shaped principal answered for an autonomous machine.
		//
		// BOTH tuples are required to reproduce it. The delegated caller is
		// still subject to perms(agent) ∩ perms(subject), so granting only the
		// user half is denied by the AGENT half for an unrelated reason — a
		// test that writes one tuple passes against the vulnerable code and
		// proves nothing. The reporter's PoC wrote both; so does this.
		setAdminCookie(t, ts)
		_, wErr := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "user:" + agent.ID, Relation: "viewer", Object: "document:readme"},
				{User: "agent:" + agent.ClientID, Relation: "viewer", Object: "document:readme"},
			},
		})
		require.NoError(t, wErr)

		assert.False(t, checkAs(t, delegated),
			"delegated token must resolve to service_account:<client_id>, not user:<sa-row-id>")
		assert.False(t, checkAs(t, machine),
			"the machine token was always denied; the delegated one must agree with it")
	})

	t.Run("the correct service_account grant plus an agent grant is allowed", func(t *testing.T) {
		// Proves the token still WORKS under its real identity — the fix denies
		// the laundered subject, it does not break machine delegation. Both
		// halves of perms(agent) ∩ perms(subject) must be granted.
		setAdminCookie(t, ts)
		_, wErr := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			// The agent half was already granted above; only the correct
			// service_account subject is missing.
			Tuples: []*model.FgaTupleInput{
				{User: "service_account:" + agent.ClientID, Relation: "viewer", Object: "document:readme"},
			},
		})
		require.NoError(t, wErr)

		assert.True(t, checkAs(t, delegated),
			"with both halves granted the delegated machine token must be allowed")
	})

	t.Run("dropping the agent half denies again: the intersection is still enforced", func(t *testing.T) {
		// The widening regression guard. Stamping login_method routes this
		// caller down resolveFgaCaller's MACHINE branch, which used to discard
		// actorID. If it still did, authority would collapse to
		// perms(subject) alone and this would wrongly stay allowed.
		setAdminCookie(t, ts)
		_, dErr := ts.GraphQLProvider.FgaDeleteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "agent:" + agent.ClientID, Relation: "viewer", Object: "document:readme"},
			},
		})
		require.NoError(t, dErr)

		assert.False(t, checkAs(t, delegated),
			"perms(agent) ∩ perms(subject) must still hold for a machine subject")
	})
}

// TestChainedMachineSubjectReExchange pins a BEHAVIOUR CHANGE introduced by
// stamping login_method, so it is a deliberate decision rather than a surprise.
//
// Before the fix, a machine-subject delegated token carried no login_method, so
// re-exchanging it sent token_exchange.go down the USER branch, which did
// GetUserByID(<service-account row id>), found nothing, and rejected the hop
// with "subject could not be verified". The multi-hop agent chain therefore
// only ever worked for its FIRST hop.
//
// After the fix the token names its real identity, so the hop takes the agent
// branch and succeeds — which is what the design intends. It stays bounded by
// every existing control: the subject client must still be active, scope is
// still intersected downward, and maxActChainDepth still caps the chain.
func TestChainedMachineSubjectReExchange(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	resource := "https://api.example.com/v1"
	subjectAgent, subjectSecret := newDelegationAgentFull(t, ts, "openid,email,profile")
	hop1Agent, hop1Secret := newDelegationAgentFull(t, ts, "openid,email")
	hop2Agent, hop2Secret := newDelegationAgentFull(t, ts, "openid")

	subjectMachine := agentAccessToken(t, ts, router, subjectAgent.ClientID, subjectSecret)
	hop1Actor := agentAccessToken(t, ts, router, hop1Agent.ClientID, hop1Secret)

	code, hop1Tok := exchangeTokens(t, ts, router, subjectMachine, hop1Actor,
		hop1Agent.ClientID, hop1Secret, resource)
	require.Equal(t, http.StatusOK, code)

	hop2Actor := agentAccessToken(t, ts, router, hop2Agent.ClientID, hop2Secret)
	code, hop2Tok := exchangeTokens(t, ts, router, hop1Tok, hop2Actor,
		hop2Agent.ClientID, hop2Secret, resource)
	require.Equal(t, http.StatusOK, code, "a machine-subject chain must now survive past hop 1")

	c2 := decodeJWTPayload(t, hop2Tok)
	assert.Equal(t, subjectAgent.ID, c2["sub"], "subject stays agent B across hops")
	assert.Equal(t, constants.AuthRecipeMethodServiceAccount, c2["login_method"],
		"the machine identity must survive every hop, not just the first")
	assert.ElementsMatch(t, []string{"openid"}, claimScope(t, c2),
		"attenuation still narrows monotonically down the chain")

	act2, ok := c2["act"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, hop2Agent.ClientID, act2["sub"], "immediate actor is hop 2")
	prior, ok := act2["act"].(map[string]interface{})
	require.True(t, ok, "hop 1 must remain nested beneath hop 2")
	assert.Equal(t, hop1Agent.ClientID, prior["sub"])

	t.Run("a deactivated subject stops the chain", func(t *testing.T) {
		// The control that bounds the newly-reachable path: liveness is still
		// re-checked at every hop, not just the first.
		subjectAgent.IsActive = false
		_, uErr := ts.StorageProvider.UpdateClient(context.Background(), subjectAgent)
		require.NoError(t, uErr)

		hop3Actor := agentAccessToken(t, ts, router, hop2Agent.ClientID, hop2Secret)
		code, _ := exchangeTokens(t, ts, router, hop2Tok, hop3Actor,
			hop2Agent.ClientID, hop2Secret, resource)
		assert.Equal(t, http.StatusBadRequest, code,
			"a deactivated service-account subject must not seed a further hop")
	})
}

// TestRequiredRelationsClassifiesMachineSubject covers the THIRD FGA decision
// surface. check_permissions and list_permissions both base their decision on
// resolveFgaCaller's classified subject; enforceRequiredRelations hardcoded
// "user:<id>" and threw that classification away, so a machine identity was
// answered as a human user there.
//
// It needs no delegation at all: a plain client_credentials token presented to
// validate_jwt_token with required_relations was satisfied by a tuple written
// for "user:<service-account-row-id>", while check_permissions denied the very
// same token. That is exactly the "two answers to one authority question" the
// surface's own doc comment warns about, and a gateway gating on
// required_relations would admit requests the permission API refuses.
func TestRequiredRelationsClassifiesMachineSubject(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaMachineDelegationModel})
	require.NoError(t, err)

	agent, secret := newDelegationAgentFull(t, ts, "openid,email")
	machine := agentAccessToken(t, ts, router, agent.ClientID, secret)

	gate := func(t *testing.T, object string) error {
		t.Helper()
		_, gErr := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			Token:     machine,
			TokenType: constants.TokenTypeAccessToken,
			RequiredRelations: []*model.FgaRelationInput{
				{Relation: "can_view", Object: object},
			},
		})
		return gErr
	}

	t.Run("a user:<sa-row-id> tuple must NOT satisfy the gate", func(t *testing.T) {
		setAdminCookie(t, ts)
		_, wErr := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "user:" + agent.ID, Relation: "viewer", Object: "document:laundered"},
			},
		})
		require.NoError(t, wErr)

		assert.Error(t, gate(t, "document:laundered"),
			"required_relations must classify the machine token as service_account:<client_id>")
	})

	t.Run("the correct service_account tuple DOES satisfy the gate", func(t *testing.T) {
		// Proves the surface still works under the real identity — the fix
		// denies the laundered subject, it does not break machine callers.
		setAdminCookie(t, ts)
		_, wErr := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "service_account:" + agent.ClientID, Relation: "viewer", Object: "document:proper"},
			},
		})
		require.NoError(t, wErr)

		assert.NoError(t, gate(t, "document:proper"),
			"a machine token granted under its real subject must pass the gate")
	})

	t.Run("required_relations agrees with check_permissions", func(t *testing.T) {
		// The invariant the surface's doc comment promises. Both must give the
		// same answer for the same token, relation and object.
		presentDelegatedToken(ts, machine)
		res, cErr := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:laundered"}},
		})
		require.NoError(t, cErr)
		require.Len(t, res.Results, 1)
		assert.False(t, res.Results[0].Allowed, "check_permissions denies the laundered subject")
		assert.Error(t, gate(t, "document:laundered"), "required_relations must agree")
	})
}
