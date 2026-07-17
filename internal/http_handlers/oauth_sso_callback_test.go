package http_handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// ssoFakeStore serves a single preset flow entry via GetAndRemoveState and
// records which key was consumed (to prove single-use). Every other memory_store
// method panics via the embedded nil — the early-exit callback paths under test
// touch only GetAndRemoveState.
type ssoFakeStore struct {
	memory_store.Provider
	entries  map[string]string
	consumed []string
}

func (s *ssoFakeStore) GetAndRemoveState(key string) (string, error) {
	s.consumed = append(s.consumed, key)
	v, ok := s.entries[key]
	if !ok {
		return "", nil
	}
	delete(s.entries, key)
	return v, nil
}

func newSSOCallbackProvider(store *ssoFakeStore) *httpProvider {
	logger := zerolog.Nop()
	return &httpProvider{
		Config:       &config.Config{},
		Dependencies: Dependencies{Log: &logger, MemoryStoreProvider: store},
	}
}

func callbackCtx(orgSlug, query string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/oauth/sso/"+orgSlug+"/callback?"+query, nil)
	c.Params = gin.Params{{Key: "org_slug", Value: orgSlug}}
	return c, rec
}

// A missing state must be rejected (CSRF: no forged callback without our state).
func TestSSOCallback_MissingStateRejected(t *testing.T) {
	store := &ssoFakeStore{entries: map[string]string{}}
	h := newSSOCallbackProvider(store)
	c, rec := callbackCtx("acme", "code=abc")
	h.SSOCallbackHandler()(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// An unknown/expired/replayed state finds nothing → rejected. Also proves the
// lookup goes through the single-use GetAndRemoveState primitive.
func TestSSOCallback_UnknownStateRejected(t *testing.T) {
	store := &ssoFakeStore{entries: map[string]string{}}
	h := newSSOCallbackProvider(store)
	c, rec := callbackCtx("acme", "state=deadbeef&code=abc")
	h.SSOCallbackHandler()(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, []string{ssoFlowPrefix + "deadbeef"}, store.consumed, "state must be consumed single-use")
}

// The callback route's org_slug must match the flow that dispatched it.
func TestSSOCallback_OrgSlugMismatchRejected(t *testing.T) {
	flow := ssoFlowState{OrgSlug: "other-org", ExpectedIssuer: ssoTestIssuer}
	raw, _ := json.Marshal(flow)
	store := &ssoFakeStore{entries: map[string]string{ssoFlowPrefix + "s1": string(raw)}}
	h := newSSOCallbackProvider(store)
	c, rec := callbackCtx("acme", "state=s1&code=abc")
	h.SSOCallbackHandler()(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Mix-up defense (RFC 9207): an `iss` query parameter that disagrees with the
// dispatching connection's issuer must be rejected before any code exchange.
func TestSSOCallback_MixupIssParamRejected(t *testing.T) {
	flow := ssoFlowState{OrgSlug: "acme", ExpectedIssuer: ssoTestIssuer}
	raw, _ := json.Marshal(flow)
	store := &ssoFakeStore{entries: map[string]string{ssoFlowPrefix + "s2": string(raw)}}
	h := newSSOCallbackProvider(store)
	c, rec := callbackCtx("acme", "state=s2&code=abc&iss=https%3A%2F%2Fattacker.example.com")
	h.SSOCallbackHandler()(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "invalid_issuer", body["error"])
}

// ssoOrgStore is a minimal storage.Provider stub serving only what
// SSOCallbackHandler's connection/org lookups touch: GetTrustedIssuerByID and
// GetOrganizationByID. Every other method panics via the embedded nil.
type ssoOrgStore struct {
	storage.Provider
	conn *schemas.TrustedIssuer
	org  *schemas.Organization
}

func (s *ssoOrgStore) GetTrustedIssuerByID(_ context.Context, id string) (*schemas.TrustedIssuer, error) {
	if s.conn != nil && s.conn.ID == id {
		return s.conn, nil
	}
	return nil, errors.New("not found")
}

func (s *ssoOrgStore) GetOrganizationByID(_ context.Context, id string) (*schemas.Organization, error) {
	if s.org != nil && s.org.ID == id {
		return s.org, nil
	}
	return nil, errors.New("not found")
}

// REGRESSION (org-disabled mid-flow race): SSOLoginHandler checks org.Enabled
// only at dispatch time (resolveActiveOIDCConnection). The callback re-fetches
// the connection by ID directly and, before this fix, never re-checked the
// owning org — so a login started before an admin disables the org still
// completed successfully within the state TTL window. Simulate that race:
// seed a flow (as SSOLoginHandler would have), then disable the org "mid-flow"
// via storage before the callback lands, and confirm the callback rejects
// instead of proceeding to code exchange / session issuance.
func TestSSOCallback_OrgDisabledMidFlightRejected(t *testing.T) {
	flow := ssoFlowState{
		ConnID:         "conn-1",
		OrgID:          "org-1",
		OrgSlug:        "acme",
		ExpectedIssuer: ssoTestIssuer,
	}
	raw, _ := json.Marshal(flow)
	memStore := &ssoFakeStore{entries: map[string]string{ssoFlowPrefix + "s3": string(raw)}}

	conn := &schemas.TrustedIssuer{ID: "conn-1", OrgID: "org-1", Kind: constants.TrustKindSSOOIDC, IsActive: true}
	// Org was enabled when SSOLoginHandler dispatched the flow, but an admin
	// disabled it before the callback arrived.
	org := &schemas.Organization{ID: "org-1", Name: "acme", Enabled: false}

	h := newSSOCallbackProvider(memStore)
	h.StorageProvider = &ssoOrgStore{conn: conn, org: org}

	c, rec := callbackCtx("acme", "state=s3&code=abc")
	h.SSOCallbackHandler()(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code, "a disabled org must reject the callback, not issue a session")
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "sso_not_configured", body["error"])
}

// Sanity counterpart: the same setup with the org still enabled must pass the
// org-enabled gate and proceed past it (fails later on the secret decrypt
// step since conn.SSOClientSecretEnc is empty here — that's fine, it proves
// the org check itself did not fire).
func TestSSOCallback_OrgEnabledPassesOrgGate(t *testing.T) {
	flow := ssoFlowState{
		ConnID:         "conn-1",
		OrgID:          "org-1",
		OrgSlug:        "acme",
		ExpectedIssuer: ssoTestIssuer,
	}
	raw, _ := json.Marshal(flow)
	memStore := &ssoFakeStore{entries: map[string]string{ssoFlowPrefix + "s4": string(raw)}}

	conn := &schemas.TrustedIssuer{ID: "conn-1", OrgID: "org-1", Kind: constants.TrustKindSSOOIDC, IsActive: true}
	org := &schemas.Organization{ID: "org-1", Name: "acme", Enabled: true}

	h := newSSOCallbackProvider(memStore)
	h.StorageProvider = &ssoOrgStore{conn: conn, org: org}
	h.Config = &config.Config{}

	c, rec := callbackCtx("acme", "state=s4&code=abc")
	h.SSOCallbackHandler()(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.NotEqual(t, "sso_not_configured", body["error"], "an enabled org must not be rejected by the org gate")
}
