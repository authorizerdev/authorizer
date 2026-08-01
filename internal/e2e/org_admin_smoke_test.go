//go:build smoke

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// TestReleaseSmokeOrgAdmin is the black-box counterpart to the in-process
// cross-surface conformance tests: it boots the real binary and asserts that an
// ORG-ADMIN — not a platform super-admin — is served the same answer on
// GraphQL, REST, and gRPC.
//
// Why this needs a real binary rather than only the in-process suite: the admin
// surface accepts two identities, and until recently only one of them worked
// off-process. The gRPC auth interceptor required a platform super-admin for
// the entire AuthorizerAdminService, so an org-admin listing their own org's
// members succeeded on GraphQL and came back Unauthenticated on gRPC and REST —
// for exactly the persona multi-tenant SSO exists to serve. That is an
// authorization decision spanning the interceptor, the gateway, and the service
// layer, wired together only in a real process.
//
// The negative cases carry as much weight as the positive one: simply removing
// the interceptor's check would satisfy the positive case alone. An org-admin
// must still be refused another org's data, and refused the platform-wide
// operations that remain super-admin-only.
func TestReleaseSmokeOrgAdmin(t *testing.T) {
	bin := buildBinary(t)
	dbPath := t.TempDir() + "/orgadmin.db"
	httpPort, metricsPort, grpcPort := freePort(t), freePort(t), freePort(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)

	startServer(t, bin, []string{
		"--database-type=sqlite", "--database-url=" + dbPath,
		"--jwt-type=HS256", "--jwt-secret=" + smokeJWTSecret,
		"--admin-secret=" + smokeAdminSecret,
		"--client-id=" + smokeClientID, "--client-secret=" + smokeClientSecret,
		fmt.Sprintf("--http-port=%d", httpPort),
		fmt.Sprintf("--metrics-port=%d", metricsPort),
		fmt.Sprintf("--grpc-port=%d", grpcPort),
		// Signup must return a usable token directly; MFA would withhold it
		// behind the setup gate. Same rationale as TestReleaseSmoke.
		"--disable-mfa",
	}, baseURL)

	// restJSON always decodes the response body, so every call needs a target
	// even when only the status code is asserted.
	var discard map[string]any

	gql := newGraphQLClient(t, baseURL)
	gql.mutate(t, `mutation { _admin_login(params:{admin_secret:"`+smokeAdminSecret+`"}) { message } }`)

	orgA := createSmokeOrg(t, gql, "smoke-org-a")
	orgB := createSmokeOrg(t, gql, "smoke-org-b")

	const email = "orgadmin@smoke.authorizer.dev"
	const password = "Password@123"
	signup := gql.mutate(t, `mutation { signup(params:{email:"`+email+`", password:"`+password+`", confirm_password:"`+password+`"}) { access_token user { id } } }`)
	token := signup["signup"].(map[string]any)["access_token"].(string)
	userID := signup["signup"].(map[string]any)["user"].(map[string]any)["id"].(string)
	require.NotEmpty(t, token)
	require.NotEmpty(t, userID)

	// Grant org-admin on A only. "authorizer:org_admin" is the reserved
	// namespaced role requireOrgAdmin accepts; the bare "admin" role does not
	// grant org-scoped admin rights.
	gql.mutate(t, `mutation { _add_org_member(params:{org_id:"`+orgA+`", user_id:"`+userID+`", roles:["authorizer:org_admin"]}) { id } }`)

	grpcConn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer grpcConn.Close()
	adminClient := authorizerv1.NewAuthorizerAdminServiceClient(grpcConn)

	// A pure-gRPC caller carries the bearer plus the authorizer host, which is
	// the issuer its token was minted with.
	bearerCtx := func(t *testing.T, bearer string) (context.Context, context.CancelFunc) {
		t.Helper()
		ctx := metadata.AppendToOutgoingContext(context.Background(),
			"authorization", "Bearer "+bearer,
			"x-authorizer-url", baseURL)
		return context.WithTimeout(ctx, 10*time.Second)
	}

	t.Run("org-admin reads their own org on every surface", func(t *testing.T) {
		data := gql.query(t, token, `query { _org_members(params:{org_id:"`+orgA+`"}) { org_members { user_id } } }`)
		require.NotNil(t, data["_org_members"], "GraphQL refused an org-admin their own org")

		ctx, cancel := bearerCtx(t, token)
		defer cancel()
		res, err := adminClient.OrgMembers(ctx, &authorizerv1.OrgMembersRequest{OrgId: orgA})
		require.NoError(t, err, "gRPC refused an org-admin their own org")
		require.NotNil(t, res)

		code := restJSON(t, baseURL, "/v1/admin/org_members", token, `{"org_id":"`+orgA+`"}`, &discard)
		assert.Equal(t, http.StatusOK, code, "REST refused an org-admin their own org")
	})

	t.Run("org-admin is refused another org on every surface", func(t *testing.T) {
		ctx, cancel := bearerCtx(t, token)
		defer cancel()
		_, err := adminClient.OrgMembers(ctx, &authorizerv1.OrgMembersRequest{OrgId: orgB})
		require.Error(t, err, "gRPC leaked another org's members")
		assert.Contains(t, []codes.Code{codes.Unauthenticated, codes.PermissionDenied}, status.Code(err))

		code := restJSON(t, baseURL, "/v1/admin/org_members", token, `{"org_id":"`+orgB+`"}`, &discard)
		assert.NotEqual(t, http.StatusOK, code, "REST leaked another org's members")
	})

	t.Run("org-admin is refused platform-wide operations on every surface", func(t *testing.T) {
		// Listing every organization is requireSuperAdmin, not requireOrgAdmin.
		ctx, cancel := bearerCtx(t, token)
		defer cancel()
		_, err := adminClient.Organizations(ctx, &authorizerv1.OrganizationsRequest{})
		require.Error(t, err, "gRPC let an org-admin list every organization")
		assert.Contains(t, []codes.Code{codes.Unauthenticated, codes.PermissionDenied}, status.Code(err))

		code := restJSON(t, baseURL, "/v1/admin/organizations", token, `{}`, &discard)
		assert.NotEqual(t, http.StatusOK, code, "REST let an org-admin list every organization")
	})

	t.Run("admin surface stays closed to anonymous and plain users", func(t *testing.T) {
		// No credential at all.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := adminClient.OrgMembers(ctx, &authorizerv1.OrgMembersRequest{OrgId: orgA})
		require.Error(t, err, "gRPC served an admin RPC with no credential")
		assert.Equal(t, codes.Unauthenticated, status.Code(err))

		code := restJSON(t, baseURL, "/v1/admin/org_members", "", `{"org_id":"`+orgA+`"}`, &discard)
		assert.Equal(t, http.StatusUnauthorized, code, "REST served an admin endpoint with no credential")

		// Authenticated, but a member of nothing. The interceptor now lets any
		// authenticated caller reach the admin handlers, so this asserts the
		// service-layer gate is what refuses them.
		const plainEmail = "plain@smoke.authorizer.dev"
		plain := gql.mutate(t, `mutation { signup(params:{email:"`+plainEmail+`", password:"`+password+`", confirm_password:"`+password+`"}) { access_token } }`)
		plainToken := plain["signup"].(map[string]any)["access_token"].(string)
		require.NotEmpty(t, plainToken)

		pctx, pcancel := bearerCtx(t, plainToken)
		defer pcancel()
		_, err = adminClient.OrgMembers(pctx, &authorizerv1.OrgMembersRequest{OrgId: orgA})
		require.Error(t, err, "gRPC served an admin RPC to a user who is a member of nothing")
		assert.Contains(t, []codes.Code{codes.Unauthenticated, codes.PermissionDenied}, status.Code(err))

		code = restJSON(t, baseURL, "/v1/admin/org_members", plainToken, `{"org_id":"`+orgA+`"}`, &discard)
		assert.NotEqual(t, http.StatusOK, code, "REST served an admin endpoint to a user who is a member of nothing")
	})
}

// createSmokeOrg creates an organization over the admin GraphQL surface and
// returns its id.
func createSmokeOrg(t *testing.T, gql *graphQLClient, name string) string {
	t.Helper()
	res := gql.mutate(t, `mutation { _create_organization(params:{name:"`+name+`"}) { id } }`)
	id, _ := res["_create_organization"].(map[string]any)["id"].(string)
	require.NotEmpty(t, id, "could not create organization %s", name)
	return id
}
