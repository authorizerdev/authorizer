package scim

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	svcscim "github.com/authorizerdev/authorizer/internal/service/scim"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// fakeService is a scim.Provider stub the handler tests drive.
type fakeService struct {
	authOrg  string
	authErr  error
	created  *schemas.User
	existed  bool
	getErr   error
	setErr   error
	lastOrg  string
	lastUser string

	createdGroup *schemas.ScimGroup
	groupErr     error
	groupMembers []string
	lastGroup    string

	listResult []*schemas.User
	listErr    error
	lastFilter svcscim.UserFilter
	lastPatch  svcscim.UserPatch
}

func (f *fakeService) Authenticate(_ context.Context, bearer string) (string, error) {
	if f.authErr != nil {
		return "", f.authErr
	}
	return f.authOrg, nil
}
func (f *fakeService) CreateUser(_ context.Context, orgID string, _ svcscim.User) (*schemas.User, bool, error) {
	f.lastOrg = orgID
	return f.created, f.existed, nil
}
func (f *fakeService) GetUser(_ context.Context, orgID, userID string) (*schemas.User, error) {
	f.lastOrg, f.lastUser = orgID, userID
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.created, nil
}
func (f *fakeService) FindByUserName(_ context.Context, orgID, _ string) (*schemas.User, error) {
	f.lastOrg = orgID
	return f.created, nil
}
func (f *fakeService) ListUsers(_ context.Context, orgID string, filter svcscim.UserFilter) ([]*schemas.User, error) {
	f.lastOrg = orgID
	f.lastFilter = filter
	return f.listResult, f.listErr
}
func (f *fakeService) PatchUser(_ context.Context, orgID, userID string, patch svcscim.UserPatch) (*schemas.User, error) {
	f.lastOrg, f.lastUser = orgID, userID
	f.lastPatch = patch
	if f.setErr != nil {
		return nil, f.setErr
	}
	return f.created, nil
}
func (f *fakeService) ReplaceUser(_ context.Context, orgID, userID string, _ svcscim.User) (*schemas.User, error) {
	f.lastOrg, f.lastUser = orgID, userID
	return f.created, f.setErr
}
func (f *fakeService) SetActive(_ context.Context, orgID, userID string, _ bool) (*schemas.User, error) {
	f.lastOrg, f.lastUser = orgID, userID
	if f.setErr != nil {
		return nil, f.setErr
	}
	return f.created, nil
}

// Group stubs — the handler-level Group tests drive the parser directly
// (parseGroupPatch); these satisfy the Provider interface for the User tests.
func (f *fakeService) CreateGroup(_ context.Context, orgID string, _ svcscim.Group) (*schemas.ScimGroup, bool, error) {
	f.lastOrg = orgID
	return f.createdGroup, f.existed, f.groupErr
}
func (f *fakeService) GetGroup(_ context.Context, orgID, groupID string) (*schemas.ScimGroup, error) {
	f.lastOrg, f.lastGroup = orgID, groupID
	return f.createdGroup, f.groupErr
}
func (f *fakeService) FindGroupByDisplayName(_ context.Context, orgID, _ string) (*schemas.ScimGroup, error) {
	f.lastOrg = orgID
	return f.createdGroup, f.groupErr
}
func (f *fakeService) ReplaceGroup(_ context.Context, orgID, groupID string, _ svcscim.Group) (*schemas.ScimGroup, error) {
	f.lastOrg, f.lastGroup = orgID, groupID
	return f.createdGroup, f.groupErr
}
func (f *fakeService) PatchGroup(_ context.Context, orgID, groupID string, _, _ *string, _ []svcscim.MemberOp) (*schemas.ScimGroup, error) {
	f.lastOrg, f.lastGroup = orgID, groupID
	return f.createdGroup, f.groupErr
}
func (f *fakeService) DeleteGroup(_ context.Context, orgID, groupID string) error {
	f.lastOrg, f.lastGroup = orgID, groupID
	return f.groupErr
}
func (f *fakeService) GroupMembers(_ context.Context, orgID, groupID string) ([]string, error) {
	f.lastOrg, f.lastGroup = orgID, groupID
	return f.groupMembers, nil
}

func newTestServer(svc svcscim.Provider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := zerolog.Nop()
	r := gin.New()
	New(&Dependencies{Log: &log, Service: svc}).Register(r.Group("/scim/v2"))
	return r
}

func do(r *gin.Engine, method, path, bearer, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func strptr(s string) *string { return &s }

func TestBadBearerReturns401WithSCIMError(t *testing.T) {
	r := newTestServer(&fakeService{authErr: svcscim.ErrUnauthorized})
	for _, tc := range []struct{ name, bearer string }{
		{"missing", ""},
		{"wrong", "ep-a.bad"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := do(r, http.MethodGet, "/scim/v2/Users/u1", tc.bearer, "")
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/scim+json")
			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			schemas, _ := body["schemas"].([]any)
			require.Len(t, schemas, 1)
			assert.Equal(t, "urn:ietf:params:scim:api:messages:2.0:Error", schemas[0])
			assert.Equal(t, "401", body["status"])
		})
	}
}

// H6 at the transport boundary: a cross-org id resolves to the service's
// ErrNotFound → 404, and the org handed to the service came from the token, not
// the URL.
func TestCrossOrgGetMapsTo404(t *testing.T) {
	svc := &fakeService{authOrg: "org-a", getErr: svcscim.ErrNotFound}
	r := newTestServer(svc)
	w := do(r, http.MethodGet, "/scim/v2/Users/victim-in-org-b", "ep.secret", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "org-a", svc.lastOrg, "org must come from the token, never the path")
}

func TestCreateUserReturns201AndSCIMShape(t *testing.T) {
	svc := &fakeService{
		authOrg: "org-a",
		created: &schemas.User{ID: "u1", Email: strptr("bob@acme.com"), ExternalID: strptr("org-a:okta-1"), IsActive: true},
	}
	r := newTestServer(svc)
	w := do(r, http.MethodPost, "/scim/v2/Users", "ep.secret",
		`{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"bob@acme.com","externalId":"okta-1","active":true}`)
	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "u1", body["id"])
	assert.Equal(t, "bob@acme.com", body["userName"])
	// externalId must be de-namespaced back to the raw IdP value.
	assert.Equal(t, "okta-1", body["externalId"])
	assert.Equal(t, true, body["active"])
}

func TestDedupCreateReturns200(t *testing.T) {
	svc := &fakeService{authOrg: "org-a", existed: true,
		created: &schemas.User{ID: "u1", Email: strptr("bob@acme.com"), IsActive: true}}
	r := newTestServer(svc)
	w := do(r, http.MethodPost, "/scim/v2/Users", "ep.secret", `{"userName":"bob@acme.com"}`)
	assert.Equal(t, http.StatusOK, w.Code, "idempotent create returns the existing resource")
}

func TestPatchActiveFalseDeactivates(t *testing.T) {
	svc := &fakeService{authOrg: "org-a",
		created: &schemas.User{ID: "u1", Email: strptr("bob@acme.com"), IsActive: false}}
	r := newTestServer(svc)
	// Both the standard and the Entra/no-path PatchOp shapes must deactivate.
	for _, body := range []string{
		`{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],"Operations":[{"op":"replace","path":"active","value":false}]}`,
		`{"Operations":[{"op":"Replace","value":{"active":false}}]}`,
	} {
		w := do(r, http.MethodPatch, "/scim/v2/Users/u1", "ep.secret", body)
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "u1", svc.lastUser)
	}
}

func TestDeleteReturns204(t *testing.T) {
	svc := &fakeService{authOrg: "org-a", created: &schemas.User{ID: "u1", IsActive: false}}
	r := newTestServer(svc)
	w := do(r, http.MethodDelete, "/scim/v2/Users/u1", "ep.secret", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "u1", svc.lastUser)
}

func TestListUserNameFilter(t *testing.T) {
	svc := &fakeService{authOrg: "org-a",
		listResult: []*schemas.User{{ID: "u1", Email: strptr("bob@acme.com"), IsActive: true}}}
	r := newTestServer(svc)
	w := do(r, http.MethodGet, `/scim/v2/Users?filter=userName+eq+%22bob@acme.com%22`, "ep.secret", "")
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.EqualValues(t, 1, body["totalResults"])
	// The parsed filter must reach the service canonicalised.
	assert.Equal(t, svcscim.UserFilter{Attribute: "userName", Operator: "eq", Value: "bob@acme.com"}, svc.lastFilter)
}

// TestListUsersFilterParsing checks the handler canonicalises every supported
// operator/attribute and 400s on unsupported filter shapes.
func TestListUsersFilterParsing(t *testing.T) {
	ok := []struct {
		query string
		want  svcscim.UserFilter
	}{
		{`filter=userName+eq+%22a@b.com%22`, svcscim.UserFilter{Attribute: "userName", Operator: "eq", Value: "a@b.com"}},
		{`filter=emails.value+eq+%22a@b.com%22`, svcscim.UserFilter{Attribute: "emails.value", Operator: "eq", Value: "a@b.com"}},
		{`filter=name.familyName+co+%22Doe%22`, svcscim.UserFilter{Attribute: "name.familyName", Operator: "co", Value: "Doe"}},
		{`filter=name.givenName+sw+%22Jo%22`, svcscim.UserFilter{Attribute: "name.givenName", Operator: "sw", Value: "Jo"}},
		{`filter=active+eq+true`, svcscim.UserFilter{Attribute: "active", Operator: "eq", Value: "true"}},
		{`filter=externalId+pr`, svcscim.UserFilter{Attribute: "externalId", Operator: "pr"}},
	}
	for _, tc := range ok {
		t.Run(tc.query, func(t *testing.T) {
			svc := &fakeService{authOrg: "org-a"}
			r := newTestServer(svc)
			w := do(r, http.MethodGet, "/scim/v2/Users?"+tc.query, "ep.secret", "")
			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tc.want, svc.lastFilter)
		})
	}

	bad := []string{
		`filter=userName+eq+%22a%22+and+active+eq+true`,          // compound
		`filter=emails%5Btype+eq+%22work%22%5D.value+eq+%22a%22`, // value-path
		`filter=displayName+eq+%22x%22`,                          // unsupported attribute
		`filter=userName+gt+%22a%22`,                             // unsupported operator
		`filter=active+co+%22tr%22`,                              // co on boolean
	}
	for _, q := range bad {
		t.Run("bad/"+q, func(t *testing.T) {
			svc := &fakeService{authOrg: "org-a"}
			r := newTestServer(svc)
			w := do(r, http.MethodGet, "/scim/v2/Users?"+q, "ep.secret", "")
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestListUsersPagination checks startIndex/count slicing over the result set.
func TestListUsersPagination(t *testing.T) {
	svc := &fakeService{authOrg: "org-a", listResult: []*schemas.User{
		{ID: "u1", Email: strptr("a@x.com"), IsActive: true},
		{ID: "u2", Email: strptr("b@x.com"), IsActive: true},
		{ID: "u3", Email: strptr("c@x.com"), IsActive: true},
	}}
	r := newTestServer(svc)
	w := do(r, http.MethodGet, `/scim/v2/Users?filter=active+eq+true&startIndex=2&count=1`, "ep.secret", "")
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.EqualValues(t, 3, body["totalResults"])
	assert.EqualValues(t, 2, body["startIndex"])
	assert.EqualValues(t, 1, body["itemsPerPage"])
	res, _ := body["Resources"].([]any)
	require.Len(t, res, 1)
}

// TestPatchUserShapes proves both the path-qualified and no-path attribute-map
// shapes reach the service as the same parsed UserPatch, and that the existing
// active-only PATCH still works.
func TestPatchUserShapes(t *testing.T) {
	t.Run("active regression (both shapes)", func(t *testing.T) {
		for _, body := range []string{
			`{"Operations":[{"op":"replace","path":"active","value":false}]}`,
			`{"Operations":[{"op":"Replace","value":{"active":false}}]}`,
		} {
			svc := &fakeService{authOrg: "org-a", created: &schemas.User{ID: "u1", IsActive: false}}
			r := newTestServer(svc)
			w := do(r, http.MethodPatch, "/scim/v2/Users/u1", "ep.secret", body)
			require.Equal(t, http.StatusOK, w.Code)
			require.NotNil(t, svc.lastPatch.Active)
			assert.False(t, *svc.lastPatch.Active)
		}
	})

	t.Run("path-qualified name.givenName", func(t *testing.T) {
		svc := &fakeService{authOrg: "org-a", created: &schemas.User{ID: "u1", IsActive: true}}
		r := newTestServer(svc)
		w := do(r, http.MethodPatch, "/scim/v2/Users/u1", "ep.secret",
			`{"Operations":[{"op":"replace","path":"name.givenName","value":"Jonathan"}]}`)
		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, svc.lastPatch.GivenName)
		assert.Equal(t, "Jonathan", *svc.lastPatch.GivenName)
	})

	t.Run("no-path attribute map", func(t *testing.T) {
		svc := &fakeService{authOrg: "org-a", created: &schemas.User{ID: "u1", IsActive: true}}
		r := newTestServer(svc)
		w := do(r, http.MethodPatch, "/scim/v2/Users/u1", "ep.secret",
			`{"Operations":[{"op":"replace","value":{"name":{"givenName":"Jonathan","familyName":"Doe"},"externalId":"okta-9"}}]}`)
		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, svc.lastPatch.GivenName)
		assert.Equal(t, "Jonathan", *svc.lastPatch.GivenName)
		require.NotNil(t, svc.lastPatch.FamilyName)
		assert.Equal(t, "Doe", *svc.lastPatch.FamilyName)
		require.NotNil(t, svc.lastPatch.ExternalID)
		assert.Equal(t, "okta-9", *svc.lastPatch.ExternalID)
	})

	t.Run("emails path-qualified and array", func(t *testing.T) {
		svc := &fakeService{authOrg: "org-a", created: &schemas.User{ID: "u1", IsActive: true}}
		r := newTestServer(svc)
		w := do(r, http.MethodPatch, "/scim/v2/Users/u1", "ep.secret",
			`{"Operations":[{"op":"replace","path":"emails[type eq \"work\"].value","value":"new@acme.com"}]}`)
		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, svc.lastPatch.Email)
		assert.Equal(t, "new@acme.com", *svc.lastPatch.Email)

		svc2 := &fakeService{authOrg: "org-a", created: &schemas.User{ID: "u1", IsActive: true}}
		r2 := newTestServer(svc2)
		w2 := do(r2, http.MethodPatch, "/scim/v2/Users/u1", "ep.secret",
			`{"Operations":[{"op":"replace","path":"emails","value":[{"value":"primary@acme.com","primary":true},{"value":"alt@acme.com"}]}]}`)
		require.Equal(t, http.StatusOK, w2.Code)
		require.NotNil(t, svc2.lastPatch.Email)
		assert.Equal(t, "primary@acme.com", *svc2.lastPatch.Email, "primary email must win")
	})

	t.Run("unmodelled path is ignored not 400", func(t *testing.T) {
		svc := &fakeService{authOrg: "org-a", created: &schemas.User{ID: "u1", IsActive: true}}
		r := newTestServer(svc)
		w := do(r, http.MethodPatch, "/scim/v2/Users/u1", "ep.secret",
			`{"Operations":[{"op":"replace","path":"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department","value":"Eng"}]}`)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Nil(t, svc.lastPatch.GivenName)
	})
}

func TestDiscoveryEndpointsServed(t *testing.T) {
	r := newTestServer(&fakeService{authOrg: "org-a"})
	for _, path := range []string{"/scim/v2/ServiceProviderConfig", "/scim/v2/Schemas", "/scim/v2/ResourceTypes"} {
		w := do(r, http.MethodGet, path, "ep.secret", "")
		assert.Equal(t, http.StatusOK, w.Code, path)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/scim+json", path)
	}
	// ServiceProviderConfig must advertise PATCH support so IdPs deprovision.
	w := do(r, http.MethodGet, "/scim/v2/ServiceProviderConfig", "ep.secret", "")
	assert.Contains(t, w.Body.String(), `"patch":{"supported":true}`)
}

func TestDiscoveryStillRequiresAuth(t *testing.T) {
	r := newTestServer(&fakeService{authErr: svcscim.ErrUnauthorized})
	w := do(r, http.MethodGet, "/scim/v2/ServiceProviderConfig", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
