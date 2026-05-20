package integration_tests

import (
	"fmt"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestMyPermissions exercises the my_permissions query end-to-end. It seeds a
// policy graph as admin, signs up a regular user, logs them in, and asserts the
// flat (resource, scope) list returned by my_permissions matches what the
// "user" role is granted via the policy targets — and that scopes attached to
// roles the principal does not hold are excluded.
func TestMyPermissions(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Authenticate as admin to seed the FGA graph.
	adminHash, err := crypto.EncryptPassword(cfg.AdminSecret)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, adminHash))

	docs, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{Name: "docs"})
	require.NoError(t, err)
	billing, err := ts.GraphQLProvider.AddResource(ctx, &model.AddResourceInput{Name: "billing"})
	require.NoError(t, err)

	read, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{Name: "read"})
	require.NoError(t, err)
	write, err := ts.GraphQLProvider.AddScope(ctx, &model.AddScopeInput{Name: "write"})
	require.NoError(t, err)

	userPolicy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name: "my-perms-user-role",
		Type: "role",
		Targets: []*model.PolicyTargetInput{
			{TargetType: "role", TargetValue: "user"},
		},
	})
	require.NoError(t, err)
	adminPolicy, err := ts.GraphQLProvider.AddPolicy(ctx, &model.AddPolicyInput{
		Name: "my-perms-admin-role",
		Type: "role",
		Targets: []*model.PolicyTargetInput{
			{TargetType: "role", TargetValue: "admin"},
		},
	})
	require.NoError(t, err)

	// user role can read docs and read billing; admin role can write docs.
	// The signed-up user has role "user" only, so they must see exactly two
	// (resource, scope) pairs and NOT docs:write.
	_, err = ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       "my-perms-docs-read",
		ResourceID: docs.ID,
		ScopeIds:   []string{read.ID},
		PolicyIds:  []string{userPolicy.ID},
	})
	require.NoError(t, err)
	_, err = ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       "my-perms-billing-read",
		ResourceID: billing.ID,
		ScopeIds:   []string{read.ID},
		PolicyIds:  []string{userPolicy.ID},
	})
	require.NoError(t, err)
	_, err = ts.GraphQLProvider.AddPermission(ctx, &model.AddPermissionInput{
		Name:       "my-perms-docs-write",
		ResourceID: docs.ID,
		ScopeIds:   []string{write.ID},
		PolicyIds:  []string{adminPolicy.ID},
	})
	require.NoError(t, err)

	req.Header.Del("Cookie")

	password := "Password@123"
	signupEmail := "my_perms_" + uuid.New().String() + "@authorizer.dev"
	_, err = ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &signupEmail,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	_, err = ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &signupEmail,
		Password: password,
	})
	require.NoError(t, err)

	// Use the freshly minted session cookie so my_permissions resolves the
	// caller via the standard session-cookie path.
	sessionToken, _ := captureTokens(t, ts)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AppCookieName+"_session", sessionToken))
	t.Cleanup(func() { req.Header.Del("Cookie") })

	perms, err := ts.GraphQLProvider.MyPermissions(ctx)
	require.NoError(t, err)
	require.NotNil(t, perms)

	got := make([]string, 0, len(perms))
	for _, p := range perms {
		require.NotNil(t, p)
		got = append(got, p.Resource+":"+p.Scope)
	}
	sort.Strings(got)

	want := []string{"billing:read", "docs:read"}
	assert.Equal(t, want, got, "my_permissions must return the user-role grants and exclude admin-only docs:write")
}
