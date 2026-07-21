package integration_tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	scimhttp "github.com/authorizerdev/authorizer/internal/http_handlers/scim"
	"github.com/authorizerdev/authorizer/internal/service/scim"
)

// scimGroupsModel is the minimal ReBAC model SCIM group membership needs: a
// group's members (and a role's assignees) may be users or nested-group usersets.
const scimGroupsModel = `model
  schema 1.1
type user
type group
  relations
    define member: [user, group#member]
type role
  relations
    define assignee: [user, group#member]
`

// bootSCIMServerFGA mounts the real inbound SCIM 2.0 handler wired to the given
// FGA engine (Group membership lives in FGA tuples, so the group ops need it).
// Mirrors bootSCIMServer but injects AuthzEngine.
func bootSCIMServerFGA(t *testing.T, ts *testSetup, eng engine.AuthorizationEngine) string {
	t.Helper()
	scimService := scim.New(&scim.Dependencies{
		Log:                 ts.Logger,
		StorageProvider:     ts.StorageProvider,
		MemoryStoreProvider: ts.MemoryStoreProvider,
		AuthzEngine:         eng,
	})
	scimHandler := scimhttp.New(&scimhttp.Dependencies{Log: ts.Logger, Service: scimService})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	scimHandler.Register(r.Group("/scim/v2"))
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv.URL
}

// TestSCIMGroupsHTTPLifecycle drives the whole Group HTTP surface end-to-end
// through the real handlers (create → get → list-with-filter → patch incl. the
// clear-members deprovisioning shapes → put → delete), plus externalId
// correlation and the 409-on-duplicate behaviour. These are exactly the paths
// the pre-fix code shipped broken: nothing previously exercised createGroup /
// getGroup / listGroups / writeGroup / toGroupResource as real HTTP handlers.
func TestSCIMGroupsHTTPLifecycle(t *testing.T) {
	cfg := getTestConfig()
	ts, eng := initFGATestSetup(t, cfg)
	_, err := eng.WriteModel(context.Background(), scimGroupsModel)
	require.NoError(t, err)
	_, ctx := createContext(ts)
	base := bootSCIMServerFGA(t, ts, eng)
	setAdminCookie(t, ts)

	org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
		Name: "scim-grp-" + uuid.NewString(),
	})
	require.NoError(t, err)
	orgID := org.ID

	tokResp, err := ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: orgID})
	require.NoError(t, err)
	token := tokResp.Token
	require.NotEmpty(t, token)

	// Provision two org members via the SCIM Users surface — group membership is
	// org-membership-gated, so members must first be users of the org.
	provisionUser := func(userName string) string {
		resp, decoded := scimDo(t, base, http.MethodPost, "/scim/v2/Users", token, map[string]any{
			"userName": userName, "active": true,
		})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		id, _ := decoded["id"].(string)
		require.NotEmpty(t, id)
		return id
	}
	u1 := provisionUser("g1-" + uuid.NewString() + "@authorizer.test")
	u2 := provisionUser("g2-" + uuid.NewString() + "@authorizer.test")

	memberValues := func(decoded map[string]any) []string {
		raw, _ := decoded["members"].([]any)
		out := make([]string, 0, len(raw))
		for _, m := range raw {
			if mm, ok := m.(map[string]any); ok {
				if v, ok := mm["value"].(string); ok {
					out = append(out, v)
				}
			}
		}
		return out
	}

	// --- Create ---
	resp, decoded := scimDo(t, base, http.MethodPost, "/scim/v2/Groups", token, map[string]any{
		"displayName": "Engineers",
		"externalId":  "grp-eng",
		"members":     []map[string]any{{"value": u1}},
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	groupID, _ := decoded["id"].(string)
	require.NotEmpty(t, groupID)
	assert.Equal(t, "Engineers", decoded["displayName"])
	assert.Equal(t, "grp-eng", decoded["externalId"])
	assert.ElementsMatch(t, []string{u1}, memberValues(decoded))
	// meta carries resourceType + created/lastModified (RFC 7643 §3.1).
	meta, _ := decoded["meta"].(map[string]any)
	require.NotNil(t, meta)
	assert.Equal(t, "Group", meta["resourceType"])
	assert.NotEmpty(t, meta["created"], "meta.created must be present")
	assert.NotEmpty(t, meta["lastModified"], "meta.lastModified must be present")

	// --- Get by id ---
	resp, decoded = scimDo(t, base, http.MethodGet, "/scim/v2/Groups/"+groupID, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Engineers", decoded["displayName"])
	assert.ElementsMatch(t, []string{u1}, memberValues(decoded))

	// --- List with displayName eq filter ---
	filter := url.QueryEscape(`displayName eq "Engineers"`)
	resp, decoded = scimDo(t, base, http.MethodGet, "/scim/v2/Groups?filter="+filter, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, float64(1), decoded["totalResults"])

	// displayName is caseExact:false (RFC 7644 §3.4.2.2): a case-variant filter
	// value must resolve the group actually named "Engineers".
	ciFilter := url.QueryEscape(`displayName eq "engineers"`)
	resp, decoded = scimDo(t, base, http.MethodGet, "/scim/v2/Groups?filter="+ciFilter, token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, float64(1), decoded["totalResults"], "case-insensitive displayName filter must find the group")
	ciResources, _ := decoded["Resources"].([]any)
	require.Len(t, ciResources, 1)
	ciGroup, _ := ciResources[0].(map[string]any)
	assert.Equal(t, "Engineers", ciGroup["displayName"], "must return the stored-cased displayName")
	assert.Equal(t, groupID, ciGroup["id"])

	// --- Unsupported filter → 400 invalidFilter (not an empty 200) ---
	badFilter := url.QueryEscape(`displayName sw "Eng"`)
	resp, decoded = scimDo(t, base, http.MethodGet, "/scim/v2/Groups?filter="+badFilter, token, nil)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalidFilter", decoded["scimType"])

	// --- Duplicate create (same displayName, no externalId) → 409 uniqueness ---
	resp, decoded = scimDo(t, base, http.MethodPost, "/scim/v2/Groups", token, map[string]any{
		"displayName": "Engineers",
	})
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "uniqueness", decoded["scimType"])

	// --- externalId correlation: same externalId, new displayName → update in
	// place (200 existed), NOT a duplicate row (MEDIUM 3c). ---
	resp, decoded = scimDo(t, base, http.MethodPost, "/scim/v2/Groups", token, map[string]any{
		"displayName": "Engineering",
		"externalId":  "grp-eng",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, groupID, decoded["id"], "same externalId must resolve the same group")
	assert.Equal(t, "Engineering", decoded["displayName"])
	// The old name no longer resolves; the new one does.
	resp, decoded = scimDo(t, base, http.MethodGet, "/scim/v2/Groups?filter="+url.QueryEscape(`displayName eq "Engineers"`), token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, float64(0), decoded["totalResults"], "renamed group must not still resolve by old name")

	// --- PATCH add u2 ---
	resp, decoded = scimDo(t, base, http.MethodPatch, "/scim/v2/Groups/"+groupID, token, map[string]any{
		"Operations": []map[string]any{
			{"op": "add", "path": "members", "value": []map[string]any{{"value": u2}}},
		},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.ElementsMatch(t, []string{u1, u2}, memberValues(decoded))

	// --- HIGH: clear all members via replace with an empty array ---
	resp, decoded = scimDo(t, base, http.MethodPatch, "/scim/v2/Groups/"+groupID, token, map[string]any{
		"Operations": []map[string]any{
			{"op": "replace", "path": "members", "value": []any{}},
		},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, memberValues(decoded), "replace with empty array must clear every member")
	assertNoGroupTuple(t, eng, u1, orgID, groupID)
	assertNoGroupTuple(t, eng, u2, orgID, groupID)

	// --- HIGH: re-add both, then clear via an unfiltered remove ---
	_, _ = scimDo(t, base, http.MethodPatch, "/scim/v2/Groups/"+groupID, token, map[string]any{
		"Operations": []map[string]any{
			{"op": "add", "path": "members", "value": []map[string]any{{"value": u1}, {"value": u2}}},
		},
	})
	resp, decoded = scimDo(t, base, http.MethodPatch, "/scim/v2/Groups/"+groupID, token, map[string]any{
		"Operations": []map[string]any{
			{"op": "remove", "path": "members"},
		},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, memberValues(decoded), "unfiltered remove must clear every member")
	assertNoGroupTuple(t, eng, u1, orgID, groupID)
	assertNoGroupTuple(t, eng, u2, orgID, groupID)

	// --- PUT replace whole resource: rename back + membership exactly {u1} ---
	resp, decoded = scimDo(t, base, http.MethodPut, "/scim/v2/Groups/"+groupID, token, map[string]any{
		"displayName": "Platform",
		"members":     []map[string]any{{"value": u1}},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Platform", decoded["displayName"])
	assert.ElementsMatch(t, []string{u1}, memberValues(decoded))

	// --- Unsupported PATCH path → 400 invalidPath ---
	resp, decoded = scimDo(t, base, http.MethodPatch, "/scim/v2/Groups/"+groupID, token, map[string]any{
		"Operations": []map[string]any{
			{"op": "replace", "path": "emails", "value": "x@y.com"},
		},
	})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "invalidPath", decoded["scimType"])

	// --- Delete → 204, then Get → 404 ---
	resp, _ = scimDo(t, base, http.MethodDelete, "/scim/v2/Groups/"+groupID, token, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp, _ = scimDo(t, base, http.MethodGet, "/scim/v2/Groups/"+groupID, token, nil)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// assertNoGroupTuple asserts that userID is NOT a member of group:<orgID>/<groupID>
// in the FGA graph — the exact fact a SAML assertion (assertedGroupsForOrg) and a
// group-derived JWT role read. Clearing members must delete these tuples.
func assertNoGroupTuple(t *testing.T, eng engine.AuthorizationEngine, userID, orgID, groupID string) {
	t.Helper()
	objects, err := eng.ListObjects(context.Background(), "user:"+userID, "member", "group")
	require.NoError(t, err)
	assert.NotContains(t, objects, "group:"+orgID+"/"+groupID,
		"cleared member must no longer resolve to the group in FGA")
}
