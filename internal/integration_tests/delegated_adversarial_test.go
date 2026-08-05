package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// advAgentModel declares `type agent`, which IS the operator opt-in that turns
// on the agent:<client_id> ∩ user:<sub> intersection (see service.FgaAgentSubjectType).
const advAgentModel = `model
  schema 1.1
type user
type agent
type document
  relations
    define viewer: [user, agent]
    define can_view: viewer
`

// advNoAgentModel is the same model WITHOUT the agent type — the "operator has
// not opted in" state.
const advNoAgentModel = `model
  schema 1.1
type user
type document
  relations
    define viewer: [user]
    define can_view: viewer
`

// ---------------------------------------------------------------------------
// (d) INTERSECTION BYPASS
// ---------------------------------------------------------------------------

// TestAdvIntersectionBypassViaExplicitUser attacks the delegation intersection
// in internal/service/check_permissions.go:47-52, which skips
// delegationSubjects whenever params.User is non-empty. The in-code comment
// claims an explicit `user` is "super-admin only", but
// service/fga.go resolveFgaSubject:84 also honours SELF-specification for any
// caller. A delegated agent can therefore echo back its own subject and have
// the agent:<client_id> half of the intersection dropped.
func TestAdvIntersectionBypassViaExplicitUser(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	req, ctx := createContext(ts)

	_, err := eng.WriteModel(ctx, advAgentModel)
	require.NoError(t, err)

	userID := "adv-user-" + uuid.NewString()
	agentID := "adv-agent-" + uuid.NewString()

	// The USER may view the document. The AGENT has NO grant at all.
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + userID, Relation: "viewer", Object: "document:secret"},
	}))

	meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts), Request: req}
	delegatedCtx := authctx.WithPrincipal(ctx, &authctx.Principal{
		UserID:  userID,
		ActorID: agentID,
	})

	check := []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:secret"}}

	t.Run("baseline: intersection denies the agent", func(t *testing.T) {
		res, _, err := ts.ServiceProvider.CheckPermissions(delegatedCtx, meta,
			&model.CheckPermissionsInput{Checks: check})
		require.NoError(t, err)
		require.Len(t, res.Results, 1)
		assert.False(t, res.Results[0].Allowed,
			"agent holds no tuple, so perms(agent) ∩ perms(user) must be empty")
	})

	t.Run("ATTACK: echo own subject in `user` to drop the agent check", func(t *testing.T) {
		res, _, err := ts.ServiceProvider.CheckPermissions(delegatedCtx, meta,
			&model.CheckPermissionsInput{
				Checks: check,
				User:   refs.NewStringRef("user:" + userID),
			})
		require.NoError(t, err, "self-specification is accepted by resolveFgaSubject")
		require.Len(t, res.Results, 1)
		assert.False(t, res.Results[0].Allowed,
			"BYPASS: supplying `user` equal to the caller's own subject skipped the agent half of the intersection")
	})

	t.Run("ATTACK: bare id form of own subject", func(t *testing.T) {
		// normalizeFgaSubject turns a bare id into user:<id>, so the bare form
		// also passes the self-specification gate.
		res, _, err := ts.ServiceProvider.CheckPermissions(delegatedCtx, meta,
			&model.CheckPermissionsInput{
				Checks: check,
				User:   refs.NewStringRef(userID),
			})
		require.NoError(t, err)
		require.Len(t, res.Results, 1)
		assert.False(t, res.Results[0].Allowed,
			"BYPASS: bare-id self-specification skipped the agent half of the intersection")
	})
}

// TestAdvListPermissionsHasNoIntersection attacks the OTHER authority-answering
// API. Only CheckPermissions was taught about delegation; ListPermissions
// (internal/service/list_permissions.go) still enumerates for the single
// resolved subject.
func TestAdvListPermissionsHasNoIntersection(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	req, ctx := createContext(ts)

	_, err := eng.WriteModel(ctx, advAgentModel)
	require.NoError(t, err)

	userID := "adv-lp-user-" + uuid.NewString()
	agentID := "adv-lp-agent-" + uuid.NewString()
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + userID, Relation: "viewer", Object: "document:lp1"},
	}))

	meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts), Request: req}
	delegatedCtx := authctx.WithPrincipal(ctx, &authctx.Principal{UserID: userID, ActorID: agentID})

	// Control: a plain (non-delegated) caller sees the object.
	plainCtx := authctx.WithPrincipal(ctx, &authctx.Principal{UserID: userID})
	base, _, err := ts.ServiceProvider.ListPermissions(plainCtx, meta, &model.ListPermissionsInput{
		Relation:   refs.NewStringRef("can_view"),
		ObjectType: refs.NewStringRef("document"),
	})
	require.NoError(t, err)
	require.Contains(t, base.Objects, "document:lp1", "control: the user itself can see the object")

	res, _, err := ts.ServiceProvider.ListPermissions(delegatedCtx, meta, &model.ListPermissionsInput{
		Relation:   refs.NewStringRef("can_view"),
		ObjectType: refs.NewStringRef("document"),
	})
	require.NoError(t, err)
	t.Logf("delegated ListPermissions objects: %v", res.Objects)
	assert.NotContains(t, res.Objects, "document:lp1",
		"ListPermissions must not hand a delegated agent the user's full object set when the agent has no grant")

	// Same explicit-`user` escape hatch as CheckPermissions
	// (internal/service/list_permissions.go:75).
	bypass, _, err := ts.ServiceProvider.ListPermissions(delegatedCtx, meta, &model.ListPermissionsInput{
		Relation:   refs.NewStringRef("can_view"),
		ObjectType: refs.NewStringRef("document"),
		User:       refs.NewStringRef("user:" + userID),
	})
	require.NoError(t, err)
	t.Logf("delegated ListPermissions objects WITH explicit user: %v", bypass.Objects)
	assert.NotContains(t, bypass.Objects, "document:lp1",
		"BYPASS: self-specified `user` drops the agent half of the enumeration intersection")
}

// ---------------------------------------------------------------------------
// (e) FAIL-OPEN on agent detection
// ---------------------------------------------------------------------------

// TestAdvNoAgentTypeDisablesEnforcement pins what happens when the model does
// NOT declare `type agent`: agentSubjectsEnabled returns false and the agent
// half of the intersection silently vanishes.
func TestAdvNoAgentTypeDisablesEnforcement(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	req, ctx := createContext(ts)

	_, err := eng.WriteModel(ctx, advNoAgentModel)
	require.NoError(t, err)

	userID := "adv-noagent-user-" + uuid.NewString()
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + userID, Relation: "viewer", Object: "document:na1"},
	}))

	meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts), Request: req}
	delegatedCtx := authctx.WithPrincipal(ctx, &authctx.Principal{
		UserID:  userID,
		ActorID: "adv-noagent-agent-" + uuid.NewString(),
	})

	res, _, err := ts.ServiceProvider.CheckPermissions(delegatedCtx, meta, &model.CheckPermissionsInput{
		Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:na1"}},
	})
	require.NoError(t, err)
	require.Len(t, res.Results, 1)
	assert.False(t, res.Results[0].Allowed,
		"a model with no agent type gives the agent the user's full authority (documented opt-in, pinned here)")
}

// TestAdvAgentDetectionFlipsOnModelRewrite is the cache-poisoning probe: the
// agent-detection cache is keyed on model id, so rewriting the model from
// agent-aware to agent-free (or back) must flip enforcement.
func TestAdvAgentDetectionFlipsOnModelRewrite(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	req, ctx := createContext(ts)

	userID := "adv-cache-user-" + uuid.NewString()
	agentID := "adv-cache-agent-" + uuid.NewString()

	_, err := eng.WriteModel(ctx, advAgentModel)
	require.NoError(t, err)
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + userID, Relation: "viewer", Object: "document:c1"},
	}))

	meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts), Request: req}
	delegatedCtx := authctx.WithPrincipal(ctx, &authctx.Principal{UserID: userID, ActorID: agentID})
	checks := []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:c1"}}

	// Prime the cache with agent detection ON.
	res, _, err := ts.ServiceProvider.CheckPermissions(delegatedCtx, meta, &model.CheckPermissionsInput{Checks: checks})
	require.NoError(t, err)
	require.False(t, res.Results[0].Allowed, "agent has no grant: denied")

	// Operator drops the agent type. New model id => cache must invalidate.
	_, err = eng.WriteModel(ctx, advNoAgentModel)
	require.NoError(t, err)

	res, _, err = ts.ServiceProvider.CheckPermissions(delegatedCtx, meta, &model.CheckPermissionsInput{Checks: checks})
	require.NoError(t, err)
	assert.False(t, res.Results[0].Allowed,
		"dropping `type agent` from the model turns the intersection off and re-grants the agent the user's authority")
}

// ---------------------------------------------------------------------------
// (a) AUDIENCE CONFUSION / reachability of the delegated path
// ---------------------------------------------------------------------------

// TestAdvDelegatedPathReachabilityViaRealEndpoint runs the REAL RFC 8693 token
// exchange and then presents the resulting token on the delegated validation
// path. The endpoint requires `resource` to be an absolute URI (RFC 8707 §2,
// http_handlers/authorize.go:1015) and stamps it verbatim as `aud`, while
// ValidateDelegatedAccessToken requires aud == Config.ClientID.
func TestAdvDelegatedPathReachabilityViaRealEndpoint(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	agentID, agentSecret := newDelegationAgent(t, ts, "openid,email,profile")
	subjectToken := testAccessToken(t, ts)
	actorToken := agentAccessToken(t, ts, router, agentID, agentSecret)

	exchange := func(resource string) *http.Response {
		f := url.Values{}
		f.Set("grant_type", tokenExchangeGrant)
		f.Set("subject_token", subjectToken)
		f.Set("subject_token_type", accessTokenType)
		f.Set("actor_token", actorToken)
		f.Set("actor_token_type", accessTokenType)
		f.Set("resource", resource)
		return postTokenExchange(ts, router, f, agentID, agentSecret).Result()
	}

	t.Run("resource cannot be the deployment client_id", func(t *testing.T) {
		resp := exchange(cfg.ClientID)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
			"the only aud that ValidateDelegatedAccessToken accepts is rejected at issuance")
	})

	t.Run("a genuinely issued delegated token does not authenticate here", func(t *testing.T) {
		f := url.Values{}
		f.Set("grant_type", tokenExchangeGrant)
		f.Set("subject_token", subjectToken)
		f.Set("subject_token_type", accessTokenType)
		f.Set("actor_token", actorToken)
		f.Set("actor_token_type", accessTokenType)
		f.Set("resource", "https://mcp.example.com")
		rec := postTokenExchange(ts, router, f, agentID, agentSecret)
		require.Equal(t, http.StatusOK, rec.Code, "exchange must succeed: %s", rec.Body.String())
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		delegated, _ := body["access_token"].(string)
		require.NotEmpty(t, delegated)

		gc := &gin.Context{Request: ts.GinContext.Request}
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, delegated)
		require.Error(t, err, "resource-bound token must not authenticate at authorizer")

		// And end-to-end through the real resolver.
		httpReq, _ := http.NewRequest(http.MethodPost, testAuthorizerHost(ts)+"/graphql", nil)
		httpReq.Header.Set("Authorization", "Bearer "+delegated)
		_, err = ts.TokenProvider.GetUserIDFromSessionOrAccessToken(&gin.Context{Request: httpReq})
		require.Error(t, err,
			"NO token this deployment can mint reaches the delegated path: the feature is unreachable")
	})
}

// ---------------------------------------------------------------------------
// (b)/(g) session bypass and token-type confusion on the delegated path
// ---------------------------------------------------------------------------

func TestAdvDelegatedPathRejectsNonDelegatedArtifacts(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	_ = ctx

	gc := &gin.Context{Request: ts.GinContext.Request}

	email := "adv_tt_" + uuid.NewString() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	login, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email: &email, Password: password,
		Scope: []string{"openid", "email", "profile", "offline_access"},
	})
	require.NoError(t, err)
	require.NotNil(t, login.AccessToken)
	require.NotNil(t, login.IDToken)
	require.NotNil(t, login.RefreshToken)
	userID := login.User.ID

	t.Run("id_token", func(t *testing.T) {
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, *login.IDToken)
		require.Error(t, err)
	})

	t.Run("refresh_token", func(t *testing.T) {
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, *login.RefreshToken)
		require.Error(t, err)
	})

	t.Run("first-party access token", func(t *testing.T) {
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, *login.AccessToken)
		require.Error(t, err, "no act claim: must not take the session-skipping path")
	})

	t.Run("session-revoked first-party access token does not fall through", func(t *testing.T) {
		httpReq, _ := http.NewRequest(http.MethodPost, testAuthorizerHost(ts)+"/graphql", nil)
		httpReq.Header.Set("Authorization", "Bearer "+*login.AccessToken)
		// Control: it works while the session lives.
		_, err := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(&gin.Context{Request: httpReq})
		require.NoError(t, err, "control: a live first-party token resolves")

		// Now nuke every session for the user (what logout / password reset do)
		// and re-present the still signature-valid token.
		require.NoError(t, ts.MemoryStoreProvider.DeleteAllUserSessions(userID))
		_, err = ts.TokenProvider.GetUserIDFromSessionOrAccessToken(&gin.Context{Request: httpReq})
		require.Error(t, err, "the delegated fallback must not resurrect a session-invalid first-party token")
	})

	_ = context.Background()
}

// ---------------------------------------------------------------------------
// (a) AUDIENCE CONFUSION — claim-shape variants
// ---------------------------------------------------------------------------

// advSign mints a JWT signed with the deployment's HS256 test secret so the
// signature gate passes and ONLY the claim logic under test decides the outcome.
// This models a hypothetical future minter emitting these shapes, not an
// attacker forging tokens (an attacker has no key).
func advSign(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok.Header["typ"] = "at+jwt"
	s, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

func TestAdvAudienceVariants(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef("adv_aud_" + uuid.NewString() + "@authorizer.dev"),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		Roles:           "user",
	})
	require.NoError(t, err)

	host := testAuthorizerHost(ts)
	gc := &gin.Context{Request: ts.GinContext.Request}

	base := func(aud interface{}) jwt.MapClaims {
		return jwt.MapClaims{
			"iss": host, "sub": user.ID,
			"exp": time.Now().Add(5 * time.Minute).Unix(), "iat": time.Now().Unix(),
			"jti": uuid.NewString(), "token_type": constants.TokenTypeAccessToken,
			"scope": []string{"openid"}, "client_id": "adv-agent",
			"act": map[string]interface{}{"sub": "adv-agent"},
			"aud": aud,
		}
	}

	cases := []struct {
		name string
		aud  interface{}
	}{
		{"array containing the client_id", []string{cfg.ClientID}},
		{"array of client_id plus a resource", []string{cfg.ClientID, "https://mcp.example.com"}},
		{"trailing slash", cfg.ClientID + "/"},
		{"upper case", strings.ToUpper(cfg.ClientID)},
		{"issuer URL", host},
		{"issuer URL with trailing slash", host + "/"},
		{"empty string", ""},
		{"resource indicator", "https://mcp.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tok := advSign(t, cfg.JWTSecret, base(tc.aud))
			_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
			require.Error(t, err, "aud=%v must not authenticate at authorizer's own API", tc.aud)
		})
	}

	t.Run("no aud claim at all", func(t *testing.T) {
		c := base(nil)
		delete(c, "aud")
		tok := advSign(t, cfg.JWTSecret, c)
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		require.Error(t, err, "a token with no audience must not authenticate")
	})

	t.Run("control: exact client_id is accepted", func(t *testing.T) {
		tok := advSign(t, cfg.JWTSecret, base(cfg.ClientID))
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		require.NoError(t, err, "control: the exact-match case must pass, else the suite proves nothing")
	})
}

// TestAdvEmptyClientIDAudienceGate probes the default deployment shape where the
// operator never passed --client-id (Config.ClientID defaults to "" — there is
// no generated fallback in Config.Finalize).
func TestAdvEmptyClientIDAudienceGate(t *testing.T) {
	cfg := getTestConfig()
	cfg.ClientID = ""
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef("adv_emptyaud_" + uuid.NewString() + "@authorizer.dev"),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		Roles:           "user",
	})
	require.NoError(t, err)

	host := testAuthorizerHost(ts)
	gc := &gin.Context{Request: ts.GinContext.Request}
	mk := func(aud interface{}) jwt.MapClaims {
		c := jwt.MapClaims{
			"iss": host, "sub": user.ID,
			"exp": time.Now().Add(5 * time.Minute).Unix(), "iat": time.Now().Unix(),
			"jti": uuid.NewString(), "token_type": constants.TokenTypeAccessToken,
			"act": map[string]interface{}{"sub": "adv-agent"},
		}
		if aud != nil {
			c["aud"] = aud
		}
		return c
	}

	t.Run("aud is an array", func(t *testing.T) {
		// aud, _ := res["aud"].(string) yields "" for a non-string claim, which
		// equals an empty Config.ClientID.
		tok := advSign(t, cfg.JWTSecret, mk([]string{"https://mcp.example.com"}))
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		require.Error(t, err, "an ARRAY aud must not slip past the string-typed audience comparison")
	})

	t.Run("aud is the empty string", func(t *testing.T) {
		tok := advSign(t, cfg.JWTSecret, mk(""))
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		require.Error(t, err, "an empty aud must not authenticate when Config.ClientID is unset")
	})
}

// ---------------------------------------------------------------------------
// (b) REPLAY of a delegated token after user-side revocation events
// ---------------------------------------------------------------------------

func TestAdvDelegatedTokenSurvivesLogoutAndPasswordReset(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "adv_replay_" + uuid.NewString() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	login, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	require.NoError(t, err)
	userID := login.User.ID

	// Only a token whose aud == Config.ClientID reaches this path, so mint one
	// directly (the token endpoint cannot produce this shape — see
	// TestAdvDelegatedPathReachabilityViaRealEndpoint).
	tok := mintDelegated(t, ts, userID, "adv-replay-agent", cfg.ClientID)
	gc := &gin.Context{Request: ts.GinContext.Request}
	_, err = ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
	require.NoError(t, err, "control")

	t.Run("survives logout / session wipe", func(t *testing.T) {
		require.NoError(t, ts.MemoryStoreProvider.DeleteAllUserSessions(userID))
		_, err := ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		assert.Error(t, err, "wiping every session must also stop the delegated token")
	})

	t.Run("survives password change", func(t *testing.T) {
		u, err := ts.StorageProvider.GetUserByID(ctx, userID)
		require.NoError(t, err)
		newPwd := "NewPassword@456"
		u.Password = &newPwd
		_, err = ts.StorageProvider.UpdateUser(ctx, u)
		require.NoError(t, err)
		_, err = ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		assert.Error(t, err, "a password reset must stop the delegated token")
	})

	t.Run("revoking the user DOES stop it", func(t *testing.T) {
		u, err := ts.StorageProvider.GetUserByID(ctx, userID)
		require.NoError(t, err)
		now := time.Now().Unix()
		u.RevokedTimestamp = &now
		_, err = ts.StorageProvider.UpdateUser(ctx, u)
		require.NoError(t, err)
		_, err = ts.TokenProvider.ValidateDelegatedAccessToken(gc, tok)
		require.Error(t, err, "RevokedTimestamp is the ONE revocation lever that works")
	})
}

// ---------------------------------------------------------------------------
// (c) ActorID shape — tuple/userset smuggling into the agent subject
// ---------------------------------------------------------------------------

// TestAdvActorIDUsersetSmuggling probes delegationSubjects
// (internal/service/fga_agent.go:141), which concatenates ActorID into an FGA
// subject WITHOUT the ContainsAny(":#@ \t\n") guard that machineFgaSubject
// applies at internal/service/fga.go:181.
func TestAdvActorIDUsersetSmuggling(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	req, ctx := createContext(ts)

	_, err := eng.WriteModel(ctx, advAgentModel)
	require.NoError(t, err)

	userID := "adv-inj-user-" + uuid.NewString()
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + userID, Relation: "viewer", Object: "document:inj"},
	}))

	meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts), Request: req}
	checks := []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:inj"}}

	for _, actor := range []string{
		"bot#viewer",
		"bot:extra",
		"*",
	} {
		t.Run("actor="+actor, func(t *testing.T) {
			dctx := authctx.WithPrincipal(ctx, &authctx.Principal{UserID: userID, ActorID: actor})
			res, _, err := ts.ServiceProvider.CheckPermissions(dctx, meta,
				&model.CheckPermissionsInput{Checks: checks})
			if err != nil {
				t.Logf("engine failed closed for actor %q: %v", actor, err)
				return
			}
			require.Len(t, res.Results, 1)
			assert.False(t, res.Results[0].Allowed,
				"a malformed/wildcard actor id must never satisfy the agent half")
		})
	}
}

// TestAdvGraphQLSurfaceHasNoDelegationPrincipal attacks the wiring rather than
// the logic. authctx.Principal.ActorID is populated in exactly ONE place —
// internal/grpcsrv/interceptors/auth.go:130/163 — so only the gRPC (and
// grpc-gateway REST) surface ever sees a delegated principal. On the GraphQL
// surface the service layer resolves the caller through
// token.GetUserIDFromSessionOrAccessToken (which DOES accept a delegated token)
// but no Principal is ever put on the context, so service.delegationSubjects
// finds none and the agent half of the intersection never runs.
func TestAdvGraphQLSurfaceHasNoDelegationPrincipal(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	_, err := eng.WriteModel(ctx, advAgentModel)
	require.NoError(t, err)

	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef("adv_gql_" + uuid.NewString() + "@authorizer.dev"),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		Roles:           "user",
	})
	require.NoError(t, err)

	// The user can view the document; the agent holds nothing.
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:" + user.ID, Relation: "viewer", Object: "document:gql"},
	}))

	delegated := mintDelegated(t, ts, user.ID, "adv-gql-agent", cfg.ClientID)

	httpReq, err := http.NewRequest(http.MethodPost, testAuthorizerHost(ts)+"/graphql", nil)
	require.NoError(t, err)
	httpReq.Header.Set("Authorization", "Bearer "+delegated)
	httpReq.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))

	// Sanity: the delegated token really does authenticate, and the token layer
	// really does surface the actor.
	data, err := ts.TokenProvider.GetUserIDFromSessionOrAccessToken(&gin.Context{Request: httpReq})
	require.NoError(t, err, "the delegated token authenticates")
	require.Equal(t, user.ID, data.UserID)
	require.Equal(t, "adv-gql-agent", data.ActorID, "the actor is available at the token layer")

	// ...but the GraphQL path never turns it into an authctx.Principal, so the
	// service layer sees an ordinary user.
	meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts), Request: httpReq}
	res, _, err := ts.ServiceProvider.CheckPermissions(ctx, meta, &model.CheckPermissionsInput{
		Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:gql"}},
	})
	require.NoError(t, err)
	require.Len(t, res.Results, 1)
	assert.False(t, res.Results[0].Allowed,
		"BYPASS: on the GraphQL surface a delegated caller is evaluated as the bare user — no agent:<client_id> check runs")
}
