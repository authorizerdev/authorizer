package integration_tests

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
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
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
	"github.com/authorizerdev/authorizer/internal/refs"
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

	// ---- Admin: a fresh store (no model yet) is an empty state, NOT an error. ----
	t.Run("_fga_get_model returns empty model on a fresh store", func(t *testing.T) {
		res, err := ts.GraphQLProvider.FgaGetModel(ctx)
		require.NoError(t, err, "no model yet must be an empty state, not an error")
		require.NotNil(t, res)
		assert.Empty(t, res.ID)
		assert.Empty(t, res.Dsl)
	})

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

	// ---- Admin: a tuple that doesn't match the model gets a friendly error. ----
	t.Run("_fga_write_tuples maps model-validation errors to an actionable message", func(t *testing.T) {
		_, err := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
			Tuples: []*model.FgaTupleInput{
				{User: "user:" + userID, Relation: "owner", Object: "document:1"},
			},
		})
		require.Error(t, err, "a relation not in the model must be rejected")
		assert.Contains(t, err.Error(), "relation 'document#owner' not found",
			"the OpenFGA reason must be surfaced")
		assert.Contains(t, err.Error(), "must be defined in the active authorization model",
			"the error must tell the admin where to fix it")
		assert.NotContains(t, err.Error(), "rpc error",
			"raw gRPC internals must not leak to the client")
	})

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

	// ---- Runtime: check_permissions allow (principal pinned to the caller). ----
	t.Run("check_permissions allows owner on granted object", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:1"}}})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Results[0].Allowed)
	})

	// ---- Runtime: check_permissions deny on a non-granted object. ----
	t.Run("check_permissions denies on ungranted object", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:2"}}})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Results[0].Allowed)
	})

	// ---- Runtime: PRINCIPAL PINNING — client cannot ask about another user. ----
	t.Run("check_permissions pins principal to caller (no impersonation)", func(t *testing.T) {
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
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:3"}}})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Results[0].Allowed, "caller must not inherit another user's grant")
	})

	// ---- Runtime: list_permissions returns only the caller's objects. ----
	t.Run("list_permissions returns granted objects for caller", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ListPermissions(ctx, &model.ListPermissionsInput{
			Relation: refs.NewStringRef("can_view"), ObjectType: refs.NewStringRef("document"),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Objects, "document:1")
		assert.NotContains(t, res.Objects, "document:3")
		assert.False(t, res.Truncated)
		require.NotEmpty(t, res.Permissions)
		assert.Equal(t, "can_view", res.Permissions[0].Relation)
	})

	// ---- Runtime: list_permissions with NO filters returns every permission
	// the caller holds across all (type, relation) pairs of the model. ----
	t.Run("list_permissions without filters returns all caller permissions", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ListPermissions(ctx, &model.ListPermissionsInput{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Objects, "document:1")
		assert.NotContains(t, res.Objects, "document:3", "another user's grant must not appear")
		assert.False(t, res.Truncated)
		// The (object, relation) detail must include both the direct tuple
		// (viewer) and the computed permission (can_view) on document:1.
		rels := make([]string, 0)
		for _, p := range res.Permissions {
			if p.Object == "document:1" {
				rels = append(rels, p.Relation)
			}
		}
		assert.Contains(t, rels, "viewer")
		assert.Contains(t, rels, "can_view")
	})

	// ---- Runtime: list_permissions with a relation-only filter. ----
	t.Run("list_permissions with relation-only filter spans object types", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ListPermissions(ctx, &model.ListPermissionsInput{
			Relation: refs.NewStringRef("viewer"),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Contains(t, res.Objects, "document:1")
		for _, p := range res.Permissions {
			assert.Equal(t, "viewer", p.Relation, "relation filter must apply to every entry")
		}
	})

	// ---- Runtime: check_permissions (batch). ----
	t.Run("check_permissions (batch) positional allow/deny", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{
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
	t.Run("check_permissions super-admin may check another subject via explicit user", func(t *testing.T) {
		setAdminCookie(t, ts)
		otherUser := "user:someone-else" // granted viewer on document:3 above
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:3"}},
			User:   &otherUser,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Results[0].Allowed, "super-admin override must evaluate the supplied subject")

		// Same admin, but checking the caller's own (admin) subject on document:3 → deny.
		adminSelf := "user:does-not-have-access"
		res, err = ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:3"}},
			User:   &adminSelf,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Results[0].Allowed)
	})

	t.Run("check_permissions ordinary user supplying another subject is REJECTED", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		otherUser := "user:someone-else"
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:3"}},
			User:   &otherUser,
		})
		assert.Error(t, err, "end user must not query another subject")
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "not authorized to query authorization for another subject")
	})

	t.Run("check_permissions ordinary user may pass their OWN subject explicitly", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		// Self-specification equals the token subject, so it is honored even for
		// non-admin callers — explicit and strict.
		self := "user:" + userID
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:1"}},
			User:   &self,
		})
		require.NoError(t, err, "self-specification must be allowed")
		require.NotNil(t, res)
		assert.True(t, res.Results[0].Allowed)
	})

	t.Run("check_permissions ordinary user with no user still self-checks", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:1"}}})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Results[0].Allowed, "self-check on a granted object must still succeed")
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
		assert.NotEmpty(t, res.ID, "_fga_get_model must return the active model id")
		assert.Contains(t, res.Dsl, "document")
	})

	// ---- Trust gate is enforced per decision op, not only on check_permissions. ----
	t.Run("list_permissions rejects ordinary user supplying another subject", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		other := "user:someone-else"
		res, err := ts.GraphQLProvider.ListPermissions(ctx, &model.ListPermissionsInput{
			Relation: refs.NewStringRef("can_view"), ObjectType: refs.NewStringRef("document"), User: &other,
		})
		assert.Error(t, err, "end user must not list another subject's objects")
		assert.Nil(t, res)
	})

	t.Run("check_permissions (batch) rejects ordinary user supplying another subject", func(t *testing.T) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, sessionToken))
		other := "user:someone-else"
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{
				{Relation: "can_view", Object: "document:3"},
			},
			User: &other,
		})
		assert.Error(t, err, "end user must not check another subject")
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

	// ---- Phase 4: validate_jwt_token honors required_relations (third entry
	// point wiring the same enforceRequiredRelations helper). ----
	t.Run("validate_jwt_token honors required_relations", func(t *testing.T) {
		// Re-login for a current access token: earlier subtests perform session
		// operations that rotate/invalidate the original token (access tokens are
		// bound to the session).
		fresh, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, fresh.AccessToken, "login must return an access token")
		accessToken := *fresh.AccessToken

		res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			TokenType: constants.TokenTypeAccessToken,
			Token:     accessToken,
			RequiredRelations: []*model.FgaRelationInput{
				{Relation: "can_view", Object: "document:1"},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.IsValid)

		_, err = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			TokenType: constants.TokenTypeAccessToken,
			Token:     accessToken,
			RequiredRelations: []*model.FgaRelationInput{
				{Relation: "can_view", Object: "document:2"},
			},
		})
		assert.Error(t, err, "validate_jwt_token must fail when a required relation is unsatisfied")
	})

	// ---- Observability: the FGA resolvers feed Prometheus metrics. ----
	t.Run("fga resolvers record decision, duration and operation metrics", func(t *testing.T) {
		// Earlier subtests rotate/invalidate the original session, so re-login for
		// a fresh one and pin to it for the decision checks.
		clearCookies(ts)
		fresh, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, fresh)
		freshSession := latestAppSessionCookie(ts)
		require.NotEmpty(t, freshSession)
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Cookie", fmt.Sprintf("%s_session=%s", constants.AppCookieName, freshSession))

		allowBefore := testutil.ToFloat64(metrics.FgaChecksTotal.WithLabelValues(metrics.FgaOpCheckPermissions, metrics.FgaResultAllowed))
		denyBefore := testutil.ToFloat64(metrics.FgaChecksTotal.WithLabelValues(metrics.FgaOpCheckPermissions, metrics.FgaResultDenied))

		// document:1 is granted to the caller (allowed); document:2 is not (denied).
		_, err = ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:1"}}})
		require.NoError(t, err)
		_, err = ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:2"}}})
		require.NoError(t, err)

		assert.Equal(t, allowBefore+1,
			testutil.ToFloat64(metrics.FgaChecksTotal.WithLabelValues(metrics.FgaOpCheckPermissions, metrics.FgaResultAllowed)),
			"an allowed check must increment the allowed counter")
		assert.Equal(t, denyBefore+1,
			testutil.ToFloat64(metrics.FgaChecksTotal.WithLabelValues(metrics.FgaOpCheckPermissions, metrics.FgaResultDenied)),
			"a denied check must increment the denied counter")
		// The duration histogram has at least the 'check' series populated.
		assert.GreaterOrEqual(t, testutil.CollectAndCount(metrics.FgaCheckDuration), 1,
			"fga check duration histogram must have observations")

		// An admin operation increments the operations counter.
		setAdminCookie(t, ts)
		opBefore := testutil.ToFloat64(metrics.FgaOperationsTotal.WithLabelValues(metrics.FgaOpReadTuples, metrics.FgaResultSuccess))
		_, err = ts.GraphQLProvider.FgaReadTuples(ctx, &model.FgaReadTuplesInput{})
		require.NoError(t, err)
		assert.Equal(t, opBefore+1,
			testutil.ToFloat64(metrics.FgaOperationsTotal.WithLabelValues(metrics.FgaOpReadTuples, metrics.FgaResultSuccess)),
			"a successful admin op must increment the operations counter")
	})

	_ = req
	_ = eng
}

// TestFGADisabled asserts fail-closed behavior when no engine is configured.
func TestFGADisabled(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg) // no AuthzEngine wired
	_, ctx := createContext(ts)

	t.Run("check_permissions errors when engine not enabled", func(t *testing.T) {
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:1"}}})
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

	// The instance works normally without FGA: a session validated without any
	// required_relations succeeds even though no FGA engine is configured. This
	// is the common case for a non-OpenFGA-compatible database (mongodb,
	// dynamodb, …) started without --fga-store.
	t.Run("works without FGA: validate_session without required_relations succeeds", func(t *testing.T) {
		email := "fga_off_ok_" + uuid.New().String() + "@authorizer.dev"
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
		})
		require.NoError(t, err, "session validation must work when FGA is disabled")
		require.NotNil(t, res)
		assert.True(t, res.IsValid)
	})

	// ---- No FGA records can be created or read via the API when no engine is
	// configured (unsupported main DB without --fga-store). Every admin op —
	// including all WRITE paths — must return the not-enabled error even for a
	// super admin, which is also what makes the dashboard's Authorization tab
	// render its FgaNotEnabled state. ----
	t.Run("all admin FGA ops fail closed when engine not enabled", func(t *testing.T) {
		setAdminCookie(t, ts)
		defer clearCookies(ts)

		const notEnabled = "fine-grained authorization is not enabled"
		tuples := &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
			{User: "user:alice", Relation: "viewer", Object: "document:1"},
		}}

		ops := []struct {
			name string
			call func() error
		}{
			{"_fga_write_model", func() error {
				_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaTestModel})
				return err
			}},
			{"_fga_write_tuples", func() error {
				_, err := ts.GraphQLProvider.FgaWriteTuples(ctx, tuples)
				return err
			}},
			{"_fga_delete_tuples", func() error {
				_, err := ts.GraphQLProvider.FgaDeleteTuples(ctx, tuples)
				return err
			}},
			{"_fga_reset", func() error {
				_, err := ts.GraphQLProvider.FgaReset(ctx)
				return err
			}},
			{"_fga_get_model", func() error {
				_, err := ts.GraphQLProvider.FgaGetModel(ctx)
				return err
			}},
			{"_fga_read_tuples", func() error {
				_, err := ts.GraphQLProvider.FgaReadTuples(ctx, &model.FgaReadTuplesInput{})
				return err
			}},
			{"_fga_list_users", func() error {
				_, err := ts.GraphQLProvider.FgaListUsers(ctx, &model.FgaListUsersInput{
					Object: "document:1", Relation: "viewer", UserType: "user",
				})
				return err
			}},
			{"_fga_expand", func() error {
				_, err := ts.GraphQLProvider.FgaExpand(ctx, &model.FgaExpandInput{
					Relation: "viewer", Object: "document:1",
				})
				return err
			}},
		}
		for _, op := range ops {
			t.Run(op.name, func(t *testing.T) {
				err := op.call()
				require.Error(t, err, "%s must fail when FGA is not enabled", op.name)
				assert.Contains(t, err.Error(), notEnabled)
			})
		}
	})

	t.Run("list_permissions errors when engine not enabled", func(t *testing.T) {
		res, err := ts.GraphQLProvider.ListPermissions(ctx, &model.ListPermissionsInput{})
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "fine-grained authorization is not enabled")
	})
}

// TestFGAExplicitStoreOverrideForUnsupportedDB proves the --fga-store override
// end-to-end at the config→engine seam: a main database OpenFGA cannot use
// (mongodb) combined with explicit --fga-store/--fga-store-url flags must
// resolve to an ENABLED FGA config, and an engine built from that resolved
// config — wired exactly the way cmd/root.go does it — must serve model
// writes, tuple writes, and checks. This is what makes the dashboard's
// Authorization tab fully functional on unsupported databases.
func TestFGAExplicitStoreOverrideForUnsupportedDB(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	cfg := getTestConfigForDB(constants.DbTypeMongoDB, "mongodb://unused-host/db")
	cfg.FGAStore = "sqlite"
	cfg.FGAStoreURL = t.TempDir() + "/fga-override.db"

	store, storeURL, enabled := cfg.FGAStoreConfig()
	require.True(t, enabled, "--fga-store must enable FGA on an unsupported main DB")
	require.Equal(t, "sqlite", store)
	require.Equal(t, "file:"+cfg.FGAStoreURL, storeURL)

	// Mirror cmd/root.go's wiring of the resolved store config.
	eng, err := fgaengine.New(&fgaengine.Config{
		Store:         store,
		StoreURL:      storeURL,
		StoreName:     "override-test",
		RunMigrations: !strings.EqualFold(store, fgaengine.StoreMemory),
	}, &fgaengine.Dependencies{Log: &logger})
	require.NoError(t, err, "engine must construct from the override store")
	t.Cleanup(func() {
		if closer, ok := eng.(interface{ Close() }); ok {
			closer.Close()
		}
	})

	_, err = eng.WriteModel(ctx, fgaTestModel)
	require.NoError(t, err)
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:alice", Relation: "viewer", Object: "document:1"},
	}))

	allowed, err := eng.Check(ctx, "user:alice", "can_view", "document:1")
	require.NoError(t, err)
	assert.True(t, allowed, "FGA must be fully functional via the explicit store override")

	allowed, err = eng.Check(ctx, "user:bob", "can_view", "document:1")
	require.NoError(t, err)
	assert.False(t, allowed)
}
