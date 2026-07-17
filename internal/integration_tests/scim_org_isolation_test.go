package integration_tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	scimhttp "github.com/authorizerdev/authorizer/internal/http_handlers/scim"
	"github.com/authorizerdev/authorizer/internal/service/scim"
)

// bootSCIMServer mounts the real inbound SCIM 2.0 handler (org resolved only
// from the bearer token, design §4.4 H6) over the same storage/memory-store
// providers as the rest of the test setup, mirroring cmd/root.go's production
// wiring. Returns the base URL.
func bootSCIMServer(t *testing.T, ts *testSetup) string {
	t.Helper()
	scimService := scim.New(&scim.Dependencies{
		Log:                 ts.Logger,
		StorageProvider:     ts.StorageProvider,
		MemoryStoreProvider: ts.MemoryStoreProvider,
	})
	scimHandler := scimhttp.New(&scimhttp.Dependencies{
		Log:     ts.Logger,
		Service: scimService,
	})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	scimHandler.Register(r.Group("/scim/v2"))
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv.URL
}

// scimDo issues a raw SCIM HTTP request with the given bearer token and
// decodes the JSON response body (if any).
func scimDo(t *testing.T, base, method, path, bearer string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, base+path, reader)
	require.NoError(t, err)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	req.Header.Set("Content-Type", "application/scim+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var decoded map[string]any
	if len(raw) > 0 {
		require.NoError(t, json.Unmarshal(raw, &decoded), "body: %s", string(raw))
	}
	return resp, decoded
}

// TestSCIMOrgIsolation closes the coverage gap flagged in the SCIM design
// review (§4.4 H6): a SCIM bearer token is scoped to exactly one org
// server-side, and org_scoped_admin_test.go's confused-deputy matrix does not
// exercise the inbound SCIM REST surface. This proves org A's SCIM connection
// can never read, enumerate, modify, or deactivate a user who belongs only to
// org B via GET/PATCH/PUT/DELETE /scim/v2/Users/:id or the userName filter
// probe on /scim/v2/Users.
func TestSCIMOrgIsolation(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	base := bootSCIMServer(t, ts)

	setAdminCookie(t, ts)

	createOrg := func(prefix string) string {
		org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: prefix + "-" + uuid.NewString(),
		})
		require.NoError(t, err)
		return org.ID
	}

	mintSCIMToken := func(orgID string) string {
		resp, err := ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: orgID})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Token)
		return resp.Token
	}

	orgA := createOrg("scim-org-a")
	orgB := createOrg("scim-org-b")
	tokenA := mintSCIMToken(orgA)
	tokenB := mintSCIMToken(orgB)

	// Provision one user into each org via that org's own SCIM connection —
	// the realistic path (Okta/Entra provisioning), not a backdoor GraphQL
	// signup.
	provision := func(bearer, userName string) string {
		resp, decoded := scimDo(t, base, http.MethodPost, "/scim/v2/Users", bearer, map[string]any{
			"userName": userName,
			"active":   true,
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		id, _ := decoded["id"].(string)
		require.NotEmpty(t, id)
		return id
	}

	userAName := "usera-" + uuid.NewString() + "@authorizer.test"
	userBName := "userb-" + uuid.NewString() + "@authorizer.test"
	_ = provision(tokenA, userAName)
	userBID := provision(tokenB, userBName)

	// Sanity: org B's own token can see its own user.
	resp, decoded := scimDo(t, base, http.MethodGet, "/scim/v2/Users/"+userBID, tokenB, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, userBName, decoded["userName"])

	t.Run("GET org-B user via org-A token -> 404, no data leak", func(t *testing.T) {
		resp, decoded := scimDo(t, base, http.MethodGet, "/scim/v2/Users/"+userBID, tokenA, nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NotEqual(t, userBName, decoded["userName"])
	})

	t.Run("PUT org-B user via org-A token -> 404, no mutation", func(t *testing.T) {
		resp, _ := scimDo(t, base, http.MethodPut, "/scim/v2/Users/"+userBID, tokenA, map[string]any{
			"userName": userBName,
			"active":   false,
		})
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("PATCH (deactivate) org-B user via org-A token -> 404, no mutation", func(t *testing.T) {
		resp, _ := scimDo(t, base, http.MethodPatch, "/scim/v2/Users/"+userBID, tokenA, map[string]any{
			"Operations": []map[string]any{
				{"op": "replace", "path": "active", "value": false},
			},
		})
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("DELETE (deprovision) org-B user via org-A token -> 404, no deactivation", func(t *testing.T) {
		resp, _ := scimDo(t, base, http.MethodDelete, "/scim/v2/Users/"+userBID, tokenA, nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("userName filter probe via org-A token does not enumerate org-B user", func(t *testing.T) {
		filter := url.QueryEscape(`userName eq "` + userBName + `"`)
		resp, decoded := scimDo(t, base, http.MethodGet,
			"/scim/v2/Users?filter="+filter, tokenA, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		resources, _ := decoded["Resources"].([]any)
		require.Empty(t, resources, "org A must not be able to enumerate org B's user by userName")
		require.Equal(t, float64(0), decoded["totalResults"])
	})

	// Confirm org B's user survived every cross-org attempt untouched: still
	// active, still fetchable with org B's own token.
	resp, decoded = scimDo(t, base, http.MethodGet, "/scim/v2/Users/"+userBID, tokenB, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, userBName, decoded["userName"])
	require.Equal(t, true, decoded["active"], "org B's user must not have been deactivated by org A's cross-org attempts")
}
