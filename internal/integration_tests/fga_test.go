package integration_tests

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	fgaengine "github.com/authorizerdev/authorizer/internal/authorization/engine/openfga"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/graphql"
	"github.com/authorizerdev/authorizer/internal/http_handlers"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

// fgaTestModel is a minimal ReBAC model used to exercise the FGA GraphQL surface.
const fgaTestModel = `model
  schema 1.1
type user
type document
  relations
    define viewer: [user]
    define can_view: viewer
`

// initFGATestSetup mirrors initTestSetup but injects an embedded OpenFGA engine
// (memory store) into both the GraphQL and HTTP providers, so the runtime/admin
// FGA resolvers are routed.
func initFGATestSetup(t *testing.T, cfg *config.Config) (*testSetup, engine.AuthorizationEngine) {
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	cfg.DatabaseURL = t.TempDir() + "/authorizer_fga.db"

	storageProvider, err := storage.New(cfg, &storage.Dependencies{Log: &logger})
	require.NoError(t, err)

	authProvider, err := authenticators.New(cfg, &authenticators.Dependencies{Log: &logger, StorageProvider: storageProvider})
	require.NoError(t, err)
	emailProvider, err := email.New(cfg, &email.Dependencies{Log: &logger, StorageProvider: storageProvider})
	require.NoError(t, err)
	eventsProvider, err := events.New(cfg, &events.Dependencies{Log: &logger, StorageProvider: storageProvider})
	require.NoError(t, err)
	memoryStoreProvider, err := memory_store.New(cfg, &memory_store.Dependencies{Log: &logger})
	require.NoError(t, err)
	smsProvider, err := sms.New(cfg, &sms.Dependencies{Log: &logger})
	require.NoError(t, err)
	tokenProvider, err := token.New(cfg, &token.Dependencies{Log: &logger, MemoryStoreProvider: memoryStoreProvider})
	require.NoError(t, err)
	rateLimitProvider, err := rate_limit.New(cfg, &rate_limit.Dependencies{Log: &logger})
	require.NoError(t, err)
	oauthProvider, err := oauth.New(cfg, &oauth.Dependencies{Log: &logger})
	require.NoError(t, err)
	auditProvider := audit.New(&audit.Dependencies{Log: &logger, StorageProvider: storageProvider})

	// Embedded OpenFGA engine with an in-memory store (dev/test only).
	fgaEngine, err := fgaengine.New(
		&fgaengine.Config{Store: fgaengine.StoreMemory, StoreName: "authorizer-test"},
		&fgaengine.Dependencies{Log: &logger},
	)
	require.NoError(t, err)

	gqlProvider, err := graphql.New(cfg, &graphql.Dependencies{
		Log:                   &logger,
		AuditProvider:         auditProvider,
		AuthenticatorProvider: authProvider,
		EmailProvider:         emailProvider,
		EventsProvider:        eventsProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		SMSProvider:           smsProvider,
		StorageProvider:       storageProvider,
		TokenProvider:         tokenProvider,
		AuthzEngine:           fgaEngine,
	})
	require.NoError(t, err)

	httpProvider, err := http_handlers.New(cfg, &http_handlers.Dependencies{
		Log:                   &logger,
		AuditProvider:         auditProvider,
		AuthenticatorProvider: authProvider,
		EmailProvider:         emailProvider,
		EventsProvider:        eventsProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		SMSProvider:           smsProvider,
		StorageProvider:       storageProvider,
		TokenProvider:         tokenProvider,
		RateLimitProvider:     rateLimitProvider,
		OAuthProvider:         oauthProvider,
		AuthzEngine:           fgaEngine,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	ctx, r := gin.CreateTestContext(w)
	r.Use(httpProvider.CORSMiddleware())
	r.Use(httpProvider.ContextMiddleware())
	r.Use(httpProvider.LoggerMiddleware())
	r.POST("/graphql", httpProvider.GraphqlHandler())
	server := httptest.NewServer(r)

	t.Cleanup(func() {
		server.Close()
		if closer, ok := fgaEngine.(interface{ Close() }); ok {
			closer.Close()
		}
		if storageProvider != nil {
			if err := storageProvider.Close(); err != nil {
				t.Logf("close storage provider: %v", err)
			}
		}
	})

	return &testSetup{
		GraphQLProvider:       gqlProvider,
		HttpProvider:          httpProvider,
		HttpServer:            server,
		Config:                cfg,
		Logger:                &logger,
		GinContext:            ctx,
		StorageProvider:       storageProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		AuthenticatorProvider: authProvider,
		TokenProvider:         tokenProvider,
	}, fgaEngine
}

// setAdminCookie authenticates the current gin request as super admin.
func setAdminCookie(t *testing.T, ts *testSetup) {
	h, err := crypto.EncryptPassword(ts.Config.AdminSecret)
	require.NoError(t, err)
	ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
}

// clearCookies removes all cookies from the current gin request.
func clearCookies(ts *testSetup) {
	ts.GinContext.Request.Header.Del("Cookie")
}

func TestFGA(t *testing.T) {
	cfg := getTestConfig()
	cfg.AuthorizationEngine = "fga"
	ts, eng := initFGATestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create + log in a regular user; their token sub is the principal.
	email := "fga_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
	require.NoError(t, err)
	require.NotNil(t, loginRes)
	userID := loginRes.User.ID
	sessionToken := latestAppSessionCookie(ts)
	require.NotEmpty(t, sessionToken)

	// ---- Admin: write the authorization model. ----
	t.Run("_fga_write_model requires super admin", func(t *testing.T) {
		clearCookies(ts)
		res, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaTestModel})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	setAdminCookie(t, ts)
	modelRes, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaTestModel})
	require.NoError(t, err)
	require.NotNil(t, modelRes)
	require.NotEmpty(t, modelRes.ID)

	// ---- Admin: write tuples granting THIS user viewer on document:1 only. ----
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
		Tuples: []*model.FgaTupleInput{
			{User: "user:" + userID, Relation: "viewer", Object: "document:1"},
		},
	})
	require.NoError(t, err)

	// ---- Admin: read tuples back. ----
	t.Run("_fga_read_tuples returns written tuple", func(t *testing.T) {
		tuplesRes, err := ts.GraphQLProvider.FgaReadTuples(ctx, &model.FgaReadTuplesInput{})
		require.NoError(t, err)
		require.NotNil(t, tuplesRes)
		found := false
		for _, tup := range tuplesRes.Tuples {
			if tup.User == "user:"+userID && tup.Relation == "viewer" && tup.Object == "document:1" {
				found = true
			}
		}
		assert.True(t, found, "written tuple should be readable")
	})

	// Switch back to the user's session for runtime calls.
	// Drop the admin cookie; runtime resolvers pin the principal from the
	// session/access token, NOT the admin cookie.
	clearCookies(ts)
	ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))

	// ---- Runtime: fga_check allow (principal pinned to the caller). ----
	t.Run("fga_check allows owner on granted object", func(t *testing.T) {
		res, err := ts.GraphQLProvider.FgaCheck(ctx, &model.FgaCheckInput{
			Relation: "can_view", Object: "document:1",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Allowed)
	})

	// ---- Runtime: fga_check deny on a non-granted object. ----
	t.Run("fga_check denies on ungranted object", func(t *testing.T) {
		res, err := ts.GraphQLProvider.FgaCheck(ctx, &model.FgaCheckInput{
			Relation: "can_view", Object: "document:2",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Allowed)
	})

	// ---- Runtime: PRINCIPAL PINNING — client cannot ask about another user. ----
	t.Run("fga_check pins principal to caller (no impersonation)", func(t *testing.T) {
		// Grant a DIFFERENT user viewer on document:3.
		setAdminCookie(t, ts)
		_, err := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "user:someone-else", Relation: "viewer", Object: "document:3"},
			},
		})
		require.NoError(t, err)
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))

		// The caller (who is NOT someone-else) must be denied on document:3.
		// There is no client-supplied "user" field, so impersonation is impossible.
		res, err := ts.GraphQLProvider.FgaCheck(ctx, &model.FgaCheckInput{
			Relation: "can_view", Object: "document:3",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Allowed, "caller must not inherit another user's grant")
	})

	// ---- Runtime: fga_list_objects returns only the caller's objects. ----
	t.Run("fga_list_objects returns granted objects for caller", func(t *testing.T) {
		res, err := ts.GraphQLProvider.FgaListObjects(ctx, &model.FgaListObjectsInput{
			Relation: "can_view", ObjectType: "document",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Objects, "document:1")
		assert.NotContains(t, res.Objects, "document:3")
	})

	// ---- Runtime: fga_batch_check. ----
	t.Run("fga_batch_check positional allow/deny", func(t *testing.T) {
		res, err := ts.GraphQLProvider.FgaBatchCheck(ctx, &model.FgaBatchCheckInput{
			Checks: []*model.FgaCheckPairInput{
				{Relation: "can_view", Object: "document:1"},
				{Relation: "can_view", Object: "document:2"},
			},
		})
		require.NoError(t, err)
		require.Len(t, res.Results, 2)
		assert.True(t, res.Results[0].Allowed)
		assert.False(t, res.Results[1].Allowed)
	})

	// ---- Admin introspection: _fga_list_users ("who can access this object?"). ----
	t.Run("_fga_list_users returns expected users; non-admin rejected", func(t *testing.T) {
		// Non-admin caller (user session, no admin cookie) must be rejected.
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		res, err := ts.GraphQLProvider.FgaListUsers(ctx, &model.FgaListUsersInput{
			Object: "document:1", Relation: "viewer", UserType: "user",
		})
		assert.Error(t, err, "non-admin must be rejected")
		assert.Nil(t, res)

		// Super-admin caller gets the access graph.
		setAdminCookie(t, ts)
		res, err = ts.GraphQLProvider.FgaListUsers(ctx, &model.FgaListUsersInput{
			Object: "document:1", Relation: "viewer", UserType: "user",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Users, "user:"+userID)
	})

	// ---- Admin introspection: _fga_expand (the "why" tree). ----
	t.Run("_fga_expand returns non-empty tree; non-admin rejected", func(t *testing.T) {
		// Non-admin caller must be rejected.
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		res, err := ts.GraphQLProvider.FgaExpand(ctx, &model.FgaExpandInput{
			Relation: "viewer", Object: "document:1",
		})
		assert.Error(t, err, "non-admin must be rejected")
		assert.Nil(t, res)

		// Super-admin caller gets a non-empty tree.
		setAdminCookie(t, ts)
		res, err = ts.GraphQLProvider.FgaExpand(ctx, &model.FgaExpandInput{
			Relation: "viewer", Object: "document:1",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.Tree)
		assert.Contains(t, res.Tree, "user:"+userID)
	})

	// ---- Trust gate: explicit `user` override on the decision ops. ----
	t.Run("fga_check super-admin may check another subject via explicit user", func(t *testing.T) {
		setAdminCookie(t, ts)
		otherUser := "user:someone-else" // granted viewer on document:3 above
		res, err := ts.GraphQLProvider.FgaCheck(ctx, &model.FgaCheckInput{
			Relation: "can_view", Object: "document:3", User: &otherUser,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Allowed, "super-admin override must evaluate the supplied subject")

		// Same admin, but checking the caller's own (admin) subject on document:3 → deny.
		adminSelf := "user:does-not-have-access"
		res, err = ts.GraphQLProvider.FgaCheck(ctx, &model.FgaCheckInput{
			Relation: "can_view", Object: "document:3", User: &adminSelf,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Allowed)
	})

	t.Run("fga_check ordinary user supplying another subject is REJECTED", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		otherUser := "user:someone-else"
		res, err := ts.GraphQLProvider.FgaCheck(ctx, &model.FgaCheckInput{
			Relation: "can_view", Object: "document:3", User: &otherUser,
		})
		assert.Error(t, err, "end user must not query another subject")
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "not authorized to query authorization for another subject")
	})

	t.Run("fga_check ordinary user with no user still self-checks", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		res, err := ts.GraphQLProvider.FgaCheck(ctx, &model.FgaCheckInput{
			Relation: "can_view", Object: "document:1",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Allowed, "self-check on a granted object must still succeed")
	})

	// ---- Phase 4: validate_session honors required_relations. ----
	t.Run("validate_session passes when required relation is satisfied", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
			Cookie: sessionToken,
			RequiredRelations: []*model.FgaRelationInput{
				{Relation: "can_view", Object: "document:1"},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.IsValid)
	})

	t.Run("validate_session fails when required relation is not satisfied", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
			Cookie: sessionToken,
			RequiredRelations: []*model.FgaRelationInput{
				{Relation: "can_view", Object: "document:2"},
			},
		})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	// ---- Admin: _fga_delete_tuples removes a tuple (and is admin-gated). ----
	t.Run("_fga_delete_tuples removes a tuple; non-admin rejected", func(t *testing.T) {
		setAdminCookie(t, ts)
		writeDel := &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
			{User: "user:" + userID, Relation: "viewer", Object: "document:deletable"},
		}}
		_, err := ts.GraphQLProvider.FgaWriteTuples(ctx, writeDel)
		require.NoError(t, err)

		// Non-admin must NOT be able to delete tuples.
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		_, err = ts.GraphQLProvider.FgaDeleteTuples(ctx, writeDel)
		assert.Error(t, err, "non-admin must not delete tuples")

		// Admin deletes, then the tuple is gone.
		setAdminCookie(t, ts)
		_, err = ts.GraphQLProvider.FgaDeleteTuples(ctx, writeDel)
		require.NoError(t, err)
		tuplesRes, err := ts.GraphQLProvider.FgaReadTuples(ctx, &model.FgaReadTuplesInput{})
		require.NoError(t, err)
		for _, tup := range tuplesRes.Tuples {
			assert.NotEqual(t, "document:deletable", tup.Object, "deleted tuple must not remain")
		}
	})

	// ---- Admin: _fga_get_model returns the active model (admin-gated). ----
	t.Run("_fga_get_model returns active model; non-admin rejected", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		res, err := ts.GraphQLProvider.FgaGetModel(ctx)
		assert.Error(t, err, "non-admin must be rejected")
		assert.Nil(t, res)

		setAdminCookie(t, ts)
		res, err = ts.GraphQLProvider.FgaGetModel(ctx)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Dsl, "document")
	})

	// ---- Trust gate is enforced per decision op, not only on fga_check. ----
	t.Run("fga_list_objects rejects ordinary user supplying another subject", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		other := "user:someone-else"
		res, err := ts.GraphQLProvider.FgaListObjects(ctx, &model.FgaListObjectsInput{
			Relation: "can_view", ObjectType: "document", User: &other,
		})
		assert.Error(t, err, "end user must not list another subject's objects")
		assert.Nil(t, res)
	})

	t.Run("fga_batch_check rejects ordinary user supplying another subject", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		other := "user:someone-else"
		res, err := ts.GraphQLProvider.FgaBatchCheck(ctx, &model.FgaBatchCheckInput{
			Checks: []*model.FgaCheckPairInput{
				{Relation: "can_view", Object: "document:3", User: &other},
			},
		})
		assert.Error(t, err, "end user must not batch-check another subject")
		assert.Nil(t, res)
	})

	// ---- Phase 4: the session query honors required_relations too (separate
	// wiring of the same enforceRequiredRelations helper as validate_session). ----
	t.Run("session query honors required_relations", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		res, err := ts.GraphQLProvider.Session(ctx, &model.SessionQueryRequest{
			RequiredRelations: []*model.FgaRelationInput{{Relation: "can_view", Object: "document:1"}},
		})
		require.NoError(t, err)
		require.NotNil(t, res)

		_, err = ts.GraphQLProvider.Session(ctx, &model.SessionQueryRequest{
			RequiredRelations: []*model.FgaRelationInput{{Relation: "can_view", Object: "document:2"}},
		})
		assert.Error(t, err, "session must fail when a required relation is unsatisfied")
	})

	_ = req
	_ = eng
}

// TestFGADisabled asserts fail-closed behavior when no engine is configured.
func TestFGADisabled(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg) // no AuthzEngine wired
	_, ctx := createContext(ts)

	t.Run("fga_check errors when engine not enabled", func(t *testing.T) {
		res, err := ts.GraphQLProvider.FgaCheck(ctx, &model.FgaCheckInput{
			Relation: "can_view", Object: "document:1",
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "fine-grained authorization is not enabled")
	})

	t.Run("validate_session errors when required_relations set but engine disabled", func(t *testing.T) {
		email := "fga_disabled_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		_, err = ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		sessionToken := latestAppSessionCookie(ts)
		require.NotEmpty(t, sessionToken)

		res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
			Cookie: sessionToken,
			RequiredRelations: []*model.FgaRelationInput{
				{Relation: "can_view", Object: "document:1"},
			},
		})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
