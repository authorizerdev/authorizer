package http_handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// ssoLoginStore is a minimal storage.Provider stub serving only what
// resolveActiveOIDCConnection touches: GetOrganizationByName and
// GetTrustedIssuerByOrgIDAndKind. Every other method panics via the embedded nil.
type ssoLoginStore struct {
	storage.Provider
	orgsByName map[string]*schemas.Organization
	connsByOrg map[string]*schemas.TrustedIssuer
}

func (s *ssoLoginStore) GetOrganizationByName(_ context.Context, name string) (*schemas.Organization, error) {
	if org, ok := s.orgsByName[name]; ok {
		return org, nil
	}
	return nil, errors.New("not found")
}

func (s *ssoLoginStore) GetTrustedIssuerByOrgIDAndKind(_ context.Context, orgID, _ string) (*schemas.TrustedIssuer, error) {
	if conn, ok := s.connsByOrg[orgID]; ok {
		return conn, nil
	}
	return nil, errors.New("not found")
}

func newSSOLoginProvider(store *ssoLoginStore) *httpProvider {
	logger := zerolog.Nop()
	return &httpProvider{
		Config: &config.Config{},
		// SSOLoginHandler checks MemoryStoreProvider != nil before resolving the
		// connection; an empty fake state store is enough to clear that gate for
		// these org-resolution tests (SetState is only reached on the success path).
		Dependencies: Dependencies{Log: &logger, StorageProvider: store, MemoryStoreProvider: &ssoFakeStore{entries: map[string]string{}}},
	}
}

func loginCtx(orgSlug, query string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/oauth/sso/"+orgSlug+"/login?"+query, nil)
	c.Params = gin.Params{{Key: "org_slug", Value: orgSlug}}
	return c, rec
}

// validSSORedirectQuery is a redirect_uri that passes IsValidRedirectURI against
// the default (wildcard) AllowedOrigins: httptest.NewRequest defaults Host to
// "example.com", and the wildcard rule restricts redirects to the server's own
// hostname.
const validSSORedirectQuery = "redirect_uri=http%3A%2F%2Fexample.com%2Fapp%2Fcallback&state=xyz"

// An unknown org slug must reject with an error response, never redirect to an
// upstream IdP.
func TestSSOLogin_UnknownOrgRejected(t *testing.T) {
	store := &ssoLoginStore{orgsByName: map[string]*schemas.Organization{}}
	h := newSSOLoginProvider(store)
	c, rec := loginCtx("no-such-org", validSSORedirectQuery)
	h.SSOLoginHandler()(c)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NotEqual(t, http.StatusTemporaryRedirect, rec.Code)
}

// A disabled org must reject with an error response, never redirect to an
// upstream IdP — this is the same org.Enabled gate SSOCallbackHandler now
// re-checks mid-flow; here it's the initiation-time check that was already
// correct (resolveActiveOIDCConnection).
func TestSSOLogin_DisabledOrgRejected(t *testing.T) {
	store := &ssoLoginStore{
		orgsByName: map[string]*schemas.Organization{
			"acme": {ID: "org-1", Name: "acme", Enabled: false},
		},
	}
	h := newSSOLoginProvider(store)
	c, rec := loginCtx("acme", validSSORedirectQuery)
	h.SSOLoginHandler()(c)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotEqual(t, http.StatusTemporaryRedirect, rec.Code)
}

// An enabled org with no active sso_oidc connection must also reject, not
// redirect (distinguishes "org disabled" from "SSO not configured").
func TestSSOLogin_EnabledOrgNoConnectionRejected(t *testing.T) {
	store := &ssoLoginStore{
		orgsByName: map[string]*schemas.Organization{
			"acme": {ID: "org-1", Name: "acme", Enabled: true},
		},
		connsByOrg: map[string]*schemas.TrustedIssuer{},
	}
	h := newSSOLoginProvider(store)
	c, rec := loginCtx("acme", validSSORedirectQuery)
	h.SSOLoginHandler()(c)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.NotEqual(t, http.StatusTemporaryRedirect, rec.Code)
}

// An enabled org with an active connection passes the resolveActiveOIDCConnection
// gate and proceeds to upstream discovery, which fails fast here since
// conn.IssuerURL is not a real reachable IdP — proving the org/connection gate
// itself did not reject the request.
func TestSSOLogin_MissingRedirectURIRejected(t *testing.T) {
	store := &ssoLoginStore{orgsByName: map[string]*schemas.Organization{}}
	h := newSSOLoginProvider(store)
	c, rec := loginCtx("acme", "state=xyz")
	h.SSOLoginHandler()(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
