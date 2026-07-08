package http_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/memory_store"
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
