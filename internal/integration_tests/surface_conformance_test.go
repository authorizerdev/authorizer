package integration_tests

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/gateway"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
)

// Cross-surface conformance.
//
// Authorizer exposes the same operations three ways — GraphQL, gRPC, and REST
// (grpc-gateway over the same gRPC server). The inventory now matches: every
// live GraphQL op has an RPC. Matching inventory is not matching BEHAVIOUR
// though, and the surfaces are wired differently enough to drift:
//
//	GraphQL  resolver  -> service provider     (auth: admin cookie)
//	gRPC     handler   -> service provider     (auth: metadata secret)
//	REST     gateway   -> gRPC handler -> ...  (auth: header secret)
//
// A field renamed in a projector, an optional collapsed to a zero value, or a
// filter applied on one path only would leave all three "working" while
// returning different answers. These tests run ONE logical operation across all
// three against a SINGLE shared backend and assert the answers agree.
//
// Shared backend matters: every surface is bound to the same testSetup, so a
// row written through one must be visible through the others. That also covers
// the read-your-writes direction, which a per-surface test cannot.

// surfaces binds all three transports to one testSetup.
type surfaces struct {
	ts       *testSetup
	grpc     authorizerv1.AuthorizerAdminServiceClient
	grpcCtx  context.Context
	restURL  string
	gqlCtx   context.Context
	adminSec string
}

// newSurfaces boots the in-process gRPC server and the grpc-gateway REST mux
// against the SAME service provider the GraphQL provider uses.
func newSurfaces(t *testing.T) *surfaces {
	t.Helper()
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, gqlCtx := createContext(ts)

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             ts.Logger,
		Config:          cfg,
		ServiceProvider: ts.ServiceProvider,
		TokenProvider:   ts.TokenProvider,
	})
	require.NoError(t, err)

	lis := bufconn.Listen(1 << 20)
	t.Cleanup(func() { _ = lis.Close() })
	go func() { _ = grpcSrv.GRPCServer().Serve(lis) }()
	t.Cleanup(grpcSrv.GRPCServer().GracefulStop)

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	gw, cleanup, err := gateway.Handler(ctx, grpcSrv.GRPCServer())
	require.NoError(t, err)
	t.Cleanup(cleanup)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/v1/*path", gin.WrapH(gw))
	rest := httptest.NewServer(r)
	t.Cleanup(rest.Close)

	return &surfaces{
		ts:       ts,
		grpc:     authorizerv1.NewAuthorizerAdminServiceClient(conn),
		grpcCtx:  adminCtx(cfg.AdminSecret),
		restURL:  rest.URL,
		gqlCtx:   gqlCtx,
		adminSec: cfg.AdminSecret,
	}
}

// asAdmin re-establishes the admin cookie the GraphQL provider authenticates
// with. Other helpers mutate the shared GinContext, so it is re-applied before
// each GraphQL call rather than once.
func (s *surfaces) asAdmin(t *testing.T) {
	t.Helper()
	clearCookies(s.ts)
	s.ts.GinContext.Request.Header.Del("Authorization")
	setAdminCookie(t, s.ts)
}

// org is the normalized projection compared across surfaces. Only fields all
// three genuinely carry — timestamps are excluded because they are set by the
// write, not the read, so comparing them would test the clock.
type org struct {
	ID          string
	Name        string
	DisplayName string
	Enabled     bool
}

func fromModel(o *model.Organization) org {
	out := org{ID: o.ID, Name: o.Name, Enabled: o.Enabled}
	if o.DisplayName != nil {
		out.DisplayName = *o.DisplayName
	}
	return out
}

func fromProto(o *authorizerv1.Organization) org {
	return org{ID: o.GetId(), Name: o.GetName(), DisplayName: o.GetDisplayName(), Enabled: o.GetEnabled()}
}

// restGetOrg reads an organization over REST and normalizes it. The gateway
// emits proto JSON, so int64s arrive as strings — irrelevant here since the
// compared fields are strings and a bool.
func (s *surfaces) restGetOrg(t *testing.T, id string) (org, int) {
	t.Helper()
	var out struct {
		Organization struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			Enabled     bool   `json:"enabled"`
		} `json:"organization"`
	}
	status := adminRESTJSON(t, s.restURL, http.MethodPost, "/v1/admin/organization",
		s.adminSec, `{"id":"`+id+`"}`, &out)
	return org{
		ID: out.Organization.ID, Name: out.Organization.Name,
		DisplayName: out.Organization.DisplayName, Enabled: out.Organization.Enabled,
	}, status
}

// TestSurfaceConformanceReadParity: an organization created on one surface must
// read back identically on all three.
func TestSurfaceConformanceReadParity(t *testing.T) {
	s := newSurfaces(t)

	s.asAdmin(t)
	name := "conformance-" + uuid.NewString()
	display := "Conformance Org"
	created, err := s.ts.GraphQLProvider.CreateOrganization(s.gqlCtx, &model.CreateOrganizationRequest{
		Name: name, DisplayName: &display,
	})
	require.NoError(t, err)
	want := fromModel(created)
	require.Equal(t, name, want.Name)

	s.asAdmin(t)
	viaGQL, err := s.ts.GraphQLProvider.Organization(s.gqlCtx, &model.OrganizationRequest{ID: created.ID})
	require.NoError(t, err, "GraphQL read")

	viaGRPC, err := s.grpc.GetOrganization(s.grpcCtx, &authorizerv1.GetOrganizationRequest{Id: created.ID})
	require.NoError(t, err, "gRPC read")

	viaREST, status := s.restGetOrg(t, created.ID)
	require.Equal(t, http.StatusOK, status, "REST read")

	assert.Equal(t, want, fromModel(viaGQL), "GraphQL disagrees with the written value")
	assert.Equal(t, want, fromProto(viaGRPC.GetOrganization()), "gRPC disagrees with the written value")
	assert.Equal(t, want, viaREST, "REST disagrees with the written value")
}

// TestSurfaceConformanceWriteVisibility: a write on ANY surface must be visible
// from the other two. Catches a surface accidentally bound to different state.
func TestSurfaceConformanceWriteVisibility(t *testing.T) {
	s := newSurfaces(t)

	t.Run("written over gRPC, read everywhere", func(t *testing.T) {
		name := "grpc-write-" + uuid.NewString()
		res, err := s.grpc.CreateOrganization(s.grpcCtx, &authorizerv1.CreateOrganizationRequest{Name: name})
		require.NoError(t, err)
		id := res.GetOrganization().GetId()
		require.NotEmpty(t, id)

		s.asAdmin(t)
		gqlRes, err := s.ts.GraphQLProvider.Organization(s.gqlCtx, &model.OrganizationRequest{ID: id})
		require.NoError(t, err)
		assert.Equal(t, name, gqlRes.Name, "GraphQL cannot see the gRPC write")

		restRes, status := s.restGetOrg(t, id)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, name, restRes.Name, "REST cannot see the gRPC write")
	})

	t.Run("written over REST, read everywhere", func(t *testing.T) {
		name := "rest-write-" + uuid.NewString()
		var out struct {
			Organization struct {
				ID string `json:"id"`
			} `json:"organization"`
		}
		status := adminRESTJSON(t, s.restURL, http.MethodPost, "/v1/admin/create_organization",
			s.adminSec, `{"name":"`+name+`"}`, &out)
		require.Equal(t, http.StatusOK, status)
		id := out.Organization.ID
		require.NotEmpty(t, id)

		s.asAdmin(t)
		gqlRes, err := s.ts.GraphQLProvider.Organization(s.gqlCtx, &model.OrganizationRequest{ID: id})
		require.NoError(t, err)
		assert.Equal(t, name, gqlRes.Name, "GraphQL cannot see the REST write")

		grpcRes, err := s.grpc.GetOrganization(s.grpcCtx, &authorizerv1.GetOrganizationRequest{Id: id})
		require.NoError(t, err)
		assert.Equal(t, name, grpcRes.GetOrganization().GetName(), "gRPC cannot see the REST write")
	})
}

// TestSurfaceConformanceListParity: after seeding, a list must return the same
// SET of organizations on every surface. Compared as a set of names because
// ordering is not part of the contract.
func TestSurfaceConformanceListParity(t *testing.T) {
	s := newSurfaces(t)

	want := map[string]bool{}
	for i := 0; i < 3; i++ {
		s.asAdmin(t)
		name := "list-parity-" + uuid.NewString()
		_, err := s.ts.GraphQLProvider.CreateOrganization(s.gqlCtx, &model.CreateOrganizationRequest{Name: name})
		require.NoError(t, err)
		want[name] = true
	}

	// A page large enough to hold everything the shared setup created.
	limit := int64(100)

	s.asAdmin(t)
	gqlRes, err := s.ts.GraphQLProvider.Organizations(s.gqlCtx, &model.ListOrganizationsRequest{
		Pagination: &model.PaginationRequest{Limit: &limit},
	})
	require.NoError(t, err)
	gqlNames := map[string]bool{}
	for _, o := range gqlRes.Organizations {
		gqlNames[o.Name] = true
	}

	grpcRes, err := s.grpc.Organizations(s.grpcCtx, &authorizerv1.OrganizationsRequest{
		Pagination: &authorizerv1.PaginationRequest{Limit: limit},
	})
	require.NoError(t, err)
	grpcNames := map[string]bool{}
	for _, o := range grpcRes.GetOrganizations() {
		grpcNames[o.GetName()] = true
	}

	var restOut struct {
		Organizations []struct {
			Name string `json:"name"`
		} `json:"organizations"`
	}
	status := adminRESTJSON(t, s.restURL, http.MethodPost, "/v1/admin/organizations",
		s.adminSec, `{"pagination":{"limit":100}}`, &restOut)
	require.Equal(t, http.StatusOK, status)
	restNames := map[string]bool{}
	for _, o := range restOut.Organizations {
		restNames[o.Name] = true
	}

	for name := range want {
		assert.True(t, gqlNames[name], "GraphQL list missing %q", name)
		assert.True(t, grpcNames[name], "gRPC list missing %q", name)
		assert.True(t, restNames[name], "REST list missing %q", name)
	}
	assert.Equal(t, len(gqlNames), len(grpcNames), "gRPC returned a different number of orgs than GraphQL")
	assert.Equal(t, len(gqlNames), len(restNames), "REST returned a different number of orgs than GraphQL")
}

// TestSurfaceConformanceFailureParity: a request that must fail has to fail on
// every surface. Transports spell the failure differently (Go error / gRPC
// status / HTTP status), so the assertion is on the OUTCOME, not the wording —
// what must not happen is one surface succeeding where another refuses.
func TestSurfaceConformanceFailureParity(t *testing.T) {
	s := newSurfaces(t)
	missing := uuid.NewString()

	t.Run("unknown organization id fails everywhere", func(t *testing.T) {
		s.asAdmin(t)
		_, gqlErr := s.ts.GraphQLProvider.Organization(s.gqlCtx, &model.OrganizationRequest{ID: missing})
		assert.Error(t, gqlErr, "GraphQL accepted an unknown org id")

		_, grpcErr := s.grpc.GetOrganization(s.grpcCtx, &authorizerv1.GetOrganizationRequest{Id: missing})
		assert.Error(t, grpcErr, "gRPC accepted an unknown org id")

		var body map[string]any
		status := adminRESTJSON(t, s.restURL, http.MethodPost, "/v1/admin/organization",
			s.adminSec, `{"id":"`+missing+`"}`, &body)
		assert.NotEqual(t, http.StatusOK, status, "REST accepted an unknown org id (body: %v)", body)
	})

	t.Run("missing admin auth is refused on gRPC and REST", func(t *testing.T) {
		// GraphQL is excluded on purpose: its auth rides the shared GinContext
		// rather than a per-call credential, so "no auth" is not expressible
		// here the way it is for the other two.
		_, grpcErr := s.grpc.GetOrganization(context.Background(),
			&authorizerv1.GetOrganizationRequest{Id: missing})
		assert.Error(t, grpcErr, "gRPC served an admin RPC with no credential")

		var body map[string]any
		status := adminRESTJSON(t, s.restURL, http.MethodPost, "/v1/admin/organization",
			"", `{"id":"`+missing+`"}`, &body)
		assert.NotEqual(t, http.StatusOK, status, "REST served an admin endpoint with no credential")
	})
}

// TestSurfaceConformanceEmptyCollections pins a classic drift point: an empty
// list must be an empty list on every surface, never null and never an error.
func TestSurfaceConformanceEmptyCollections(t *testing.T) {
	s := newSurfaces(t)

	s.asAdmin(t)
	created, err := s.ts.GraphQLProvider.CreateOrganization(s.gqlCtx,
		&model.CreateOrganizationRequest{Name: "empty-" + uuid.NewString()})
	require.NoError(t, err)

	s.asAdmin(t)
	gqlRes, err := s.ts.GraphQLProvider.OrgMembers(s.gqlCtx, &model.ListOrgMembersRequest{OrgID: created.ID})
	require.NoError(t, err, "GraphQL org_members on a fresh org")
	assert.Empty(t, gqlRes.OrgMembers)

	grpcRes, err := s.grpc.OrgMembers(s.grpcCtx, &authorizerv1.OrgMembersRequest{OrgId: created.ID})
	require.NoError(t, err, "gRPC OrgMembers on a fresh org")
	assert.Empty(t, grpcRes.GetOrgMembers())

	raw := adminRESTRaw(t, s.restURL, "/v1/admin/org_members", s.adminSec, `{"org_id":"`+created.ID+`"}`)
	var restRes struct {
		OrgMembers []json.RawMessage `json:"org_members"`
	}
	require.NoError(t, json.Unmarshal(raw, &restRes), "REST org_members body: %s", raw)
	// NotNil before Empty, deliberately: assert.Empty alone is satisfied by an
	// empty slice, by JSON null, AND by the key being absent entirely, so it
	// would NOT catch the drift this test exists for. Decoding null or a missing
	// key yields a nil slice; only a literal [] yields empty-but-non-nil.
	assert.NotNil(t, restRes.OrgMembers,
		"REST must serialize an empty collection as [], not null or an absent key (body: %s)", raw)
	assert.Empty(t, restRes.OrgMembers)
}

// adminRESTRaw is adminRESTJSON's raw-body sibling. Needed where the assertion
// is about the JSON SHAPE itself — an empty list must serialize as [] and not
// null — which a decode into a typed struct would hide.
func adminRESTRaw(t *testing.T, baseURL, path, adminSecret, body string) []byte {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", baseURL)
	if adminSecret != "" {
		req.Header.Set("x-authorizer-admin-secret", adminSecret)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return raw
}
