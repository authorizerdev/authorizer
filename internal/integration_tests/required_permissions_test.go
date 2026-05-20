package integration_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureTokens returns the most recent session and access tokens from the
// in-memory store. Helper for tests that need to read tokens minted by Login.
func captureTokens(t *testing.T, ts *testSetup) (sessionToken, accessToken string) {
	t.Helper()
	allData, err := ts.MemoryStoreProvider.GetAllData()
	require.NoError(t, err)
	for k, v := range allData {
		switch {
		case strings.Contains(k, constants.TokenTypeSessionToken):
			sessionToken = v
		case strings.Contains(k, constants.TokenTypeAccessToken):
			accessToken = v
		}
	}
	require.NotEmpty(t, sessionToken, "session token must be present")
	require.NotEmpty(t, accessToken, "access token must be present")
	return sessionToken, accessToken
}

// TestRequiredPermissions verifies the new optional required_permissions field
// on session, validate_jwt_token, and validate_session. It also asserts the
// backward-compatible path (callers that omit the field see no change).
func TestRequiredPermissions(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Seed an authz permission as admin: docs:read granted to the "user"
	// role. The default signup assigns "user" role.
	adminHash, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	resource, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{Name: "docs"})
	require.NoError(t, err)

	scope, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{Name: "read"})
	require.NoError(t, err)

	policy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name: "user-role-can-read",
		Type: "role",
		Targets: []*model.PolicyTargetInput{
			{TargetType: "role", TargetValue: "user"},
		},
	})
	require.NoError(t, err)

	_, err = ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       "docs-read",
		ResourceID: resource.ID,
		ScopeIds:   []string{scope.ID},
		PolicyIds:  []string{policy.ID},
	})
	require.NoError(t, err)

	req.Header.Del("Cookie")

	password := "Password@123"
	signupEmail := "required_perms_" + uuid.New().String() + "@authorizer.dev"
	_, err = ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &signupEmail,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	login := func(t *testing.T) {
		t.Helper()
		_, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &signupEmail,
			Password: password,
		})
		require.NoError(t, err)
	}

	// validate_jwt_token and validate_session do NOT rotate, so a single login
	// suffices for all six of their subtests.
	login(t)
	sessionToken, accessToken := captureTokens(t, ts)

	t.Run("validate_jwt_token", func(t *testing.T) {
		t.Run("backward compat: no required_permissions still works", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
				Token:     accessToken,
				TokenType: constants.TokenTypeAccessToken,
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.True(t, res.IsValid)
		})

		t.Run("granted permission passes", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
				Token:     accessToken,
				TokenType: constants.TokenTypeAccessToken,
				RequiredPermissions: []*model.PermissionInput{
					{Resource: "docs", Scope: "read"},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.True(t, res.IsValid)
		})

		t.Run("denied permission returns unauthorized", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
				Token:     accessToken,
				TokenType: constants.TokenTypeAccessToken,
				RequiredPermissions: []*model.PermissionInput{
					{Resource: "docs", Scope: "write"},
				},
			})
			require.Error(t, err)
			require.Nil(t, res)
			assert.Contains(t, err.Error(), "unauthorized")
		})
	})

	t.Run("validate_session", func(t *testing.T) {
		t.Run("backward compat: no required_permissions still works", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
				Cookie: sessionToken,
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.True(t, res.IsValid)
		})

		t.Run("granted permission passes", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
				Cookie: sessionToken,
				RequiredPermissions: []*model.PermissionInput{
					{Resource: "docs", Scope: "read"},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.True(t, res.IsValid)
		})

		t.Run("denied permission returns unauthorized", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
				Cookie: sessionToken,
				RequiredPermissions: []*model.PermissionInput{
					{Resource: "docs", Scope: "write"},
				},
			})
			require.Error(t, err)
			require.Nil(t, res)
			assert.Contains(t, err.Error(), "unauthorized")
		})
	})

	// session() rotates the session on every successful call — re-login per
	// subtest so each one starts with a fresh, valid session cookie.
	callSession := func(t *testing.T, params *model.SessionQueryRequest) (*model.AuthResponse, error) {
		t.Helper()
		login(t)
		st, _ := captureTokens(t, ts)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AppCookieName+"_session", st))
		defer req.Header.Del("Cookie")
		return ts.GraphQLProvider.Session(ctx, params)
	}

	t.Run("session", func(t *testing.T) {
		t.Run("backward compat: no required_permissions still works", func(t *testing.T) {
			res, err := callSession(t, &model.SessionQueryRequest{})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.NotEmpty(t, res.AccessToken)
		})

		t.Run("granted permission passes", func(t *testing.T) {
			res, err := callSession(t, &model.SessionQueryRequest{
				RequiredPermissions: []*model.PermissionInput{
					{Resource: "docs", Scope: "read"},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.NotEmpty(t, res.AccessToken)
		})

		t.Run("denied permission returns unauthorized", func(t *testing.T) {
			res, err := callSession(t, &model.SessionQueryRequest{
				RequiredPermissions: []*model.PermissionInput{
					{Resource: "docs", Scope: "write"},
				},
			})
			require.Error(t, err)
			require.Nil(t, res)
			assert.Contains(t, err.Error(), "unauthorized")
		})
	})

	t.Run("metrics counters increment per outcome", func(t *testing.T) {
		// Re-login to get a fresh access token — the session subtests above each
		// call login() internally (session rotates on every call), which replaces
		// the memory-store entries and makes the top-level accessToken stale.
		login(t)
		_, freshAccessToken := captureTokens(t, ts)

		grantedBefore := testutil.ToFloat64(metrics.RequiredPermissionsChecksTotal.WithLabelValues(
			metrics.RequiredPermissionsEndpointValidateJWTToken,
			metrics.RequiredPermissionsOutcomeGranted,
		))
		deniedBefore := testutil.ToFloat64(metrics.RequiredPermissionsChecksTotal.WithLabelValues(
			metrics.RequiredPermissionsEndpointValidateJWTToken,
			metrics.RequiredPermissionsOutcomeDenied,
		))
		notReqBefore := testutil.ToFloat64(metrics.RequiredPermissionsChecksTotal.WithLabelValues(
			metrics.RequiredPermissionsEndpointValidateJWTToken,
			metrics.RequiredPermissionsOutcomeNotRequested,
		))

		// not_requested
		_, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			Token:     freshAccessToken,
			TokenType: constants.TokenTypeAccessToken,
		})
		require.NoError(t, err)

		// granted
		_, err = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			Token:     freshAccessToken,
			TokenType: constants.TokenTypeAccessToken,
			RequiredPermissions: []*model.PermissionInput{
				{Resource: "docs", Scope: "read"},
			},
		})
		require.NoError(t, err)

		// denied — error is intentional; only the counter increment matters.
		// outcome=error is not exercised here: simulating a CheckPermission
		// storage fault from an integration test requires fault injection
		// the provider doesn't currently expose.
		_, _ = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			Token:     freshAccessToken,
			TokenType: constants.TokenTypeAccessToken,
			RequiredPermissions: []*model.PermissionInput{
				{Resource: "docs", Scope: "write"},
			},
		})

		assert.Equal(t, grantedBefore+1, testutil.ToFloat64(metrics.RequiredPermissionsChecksTotal.WithLabelValues(
			metrics.RequiredPermissionsEndpointValidateJWTToken,
			metrics.RequiredPermissionsOutcomeGranted,
		)))
		assert.Equal(t, deniedBefore+1, testutil.ToFloat64(metrics.RequiredPermissionsChecksTotal.WithLabelValues(
			metrics.RequiredPermissionsEndpointValidateJWTToken,
			metrics.RequiredPermissionsOutcomeDenied,
		)))
		assert.Equal(t, notReqBefore+1, testutil.ToFloat64(metrics.RequiredPermissionsChecksTotal.WithLabelValues(
			metrics.RequiredPermissionsEndpointValidateJWTToken,
			metrics.RequiredPermissionsOutcomeNotRequested,
		)))
	})
}
