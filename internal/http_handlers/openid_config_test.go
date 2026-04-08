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
)

// newDiscoveryTestProvider builds a minimal httpProvider suitable for unit
// testing OpenIDConfigurationHandler. The handler only reads Config + Log,
// so no storage/memory_store dependencies are required.
func newDiscoveryTestProvider(t *testing.T, cfg *config.Config) *httpProvider {
	t.Helper()
	logger := zerolog.Nop()
	return &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log: &logger,
		},
	}
}

func doDiscoveryRequest(t *testing.T, h *httpProvider) map[string]interface{} {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/.well-known/openid-configuration", h.OpenIDConfigurationHandler())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func toStringSlice(t *testing.T, v interface{}) []string {
	t.Helper()
	raw, ok := v.([]interface{})
	require.True(t, ok, "expected JSON array, got %T", v)
	out := make([]string, len(raw))
	for i, item := range raw {
		s, ok := item.(string)
		require.True(t, ok, "expected string element, got %T", item)
		out[i] = s
	}
	return out
}

// TestDiscovery_ClaimsSupportedDoesNotContainRole asserts that "role"
// (singular) is not advertised — only "roles" is actually emitted by the
// token issuer, so advertising "role" violates OIDC Discovery §3.
func TestDiscovery_ClaimsSupportedDoesNotContainRole(t *testing.T) {
	h := newDiscoveryTestProvider(t, &config.Config{JWTType: "RS256"})
	body := doDiscoveryRequest(t, h)
	claims := toStringSlice(t, body["claims_supported"])
	assert.NotContains(t, claims, "role", "advertising 'role' violates OIDC Discovery — issuer only emits 'roles'")
}

// TestDiscovery_ClaimsSupportedContainsRoles asserts that "roles" (plural)
// IS advertised, since the token issuer emits it.
func TestDiscovery_ClaimsSupportedContainsRoles(t *testing.T) {
	h := newDiscoveryTestProvider(t, &config.Config{JWTType: "RS256"})
	body := doDiscoveryRequest(t, h)
	claims := toStringSlice(t, body["claims_supported"])
	assert.Contains(t, claims, "roles")
}

// TestDiscovery_AdvertisesPrimaryAndSecondaryAlg asserts that both the
// primary and secondary signing algorithms are advertised when an operator
// runs a key rotation pair (e.g. RS256 primary + ES256 secondary), with no
// duplicates and RS256 still present (OIDC Discovery §3 MUST).
func TestDiscovery_AdvertisesPrimaryAndSecondaryAlg(t *testing.T) {
	h := newDiscoveryTestProvider(t, &config.Config{
		JWTType:          "RS256",
		JWTSecondaryType: "ES256",
	})
	body := doDiscoveryRequest(t, h)
	algs := toStringSlice(t, body["id_token_signing_alg_values_supported"])

	assert.Contains(t, algs, "RS256")
	assert.Contains(t, algs, "ES256")

	seen := make(map[string]int, len(algs))
	for _, a := range algs {
		seen[a]++
	}
	for alg, count := range seen {
		assert.Equalf(t, 1, count, "alg %q must not be duplicated", alg)
	}
}

// TestDiscovery_AdvertisesOnlyPrimaryAlgWhenNoSecondary asserts that when
// no secondary key is configured, the primary alg is advertised (with
// RS256 still guaranteed present per OIDC Discovery §3).
func TestDiscovery_AdvertisesOnlyPrimaryAlgWhenNoSecondary(t *testing.T) {
	h := newDiscoveryTestProvider(t, &config.Config{JWTType: "RS256"})
	body := doDiscoveryRequest(t, h)
	algs := toStringSlice(t, body["id_token_signing_alg_values_supported"])

	require.Len(t, algs, 1, "RS256 primary with no secondary should advertise exactly RS256")
	assert.Equal(t, "RS256", algs[0])
}

// TestDiscovery_AdvertisesOnlyPrimaryAlgWhenNoSecondary_NonRS256 asserts
// that with a non-RS256 primary and no secondary, both the primary alg and
// the mandatory RS256 are advertised, with no duplicates.
func TestDiscovery_AdvertisesOnlyPrimaryAlgWhenNoSecondary_NonRS256(t *testing.T) {
	h := newDiscoveryTestProvider(t, &config.Config{JWTType: "HS256"})
	body := doDiscoveryRequest(t, h)
	algs := toStringSlice(t, body["id_token_signing_alg_values_supported"])

	assert.Contains(t, algs, "HS256")
	assert.Contains(t, algs, "RS256")
	assert.Len(t, algs, 2)
}

// TestDiscovery_DoesNotAdvertiseWebMessage asserts that the vendor
// extension "web_message" is not advertised in response_modes_supported —
// it is not in the IANA OAuth Authorization Endpoint Response Modes
// registry.
func TestDiscovery_DoesNotAdvertiseWebMessage(t *testing.T) {
	h := newDiscoveryTestProvider(t, &config.Config{JWTType: "RS256"})
	body := doDiscoveryRequest(t, h)
	modes := toStringSlice(t, body["response_modes_supported"])
	assert.NotContains(t, modes, "web_message", "web_message is a vendor extension and not IANA-registered")
}

// TestDiscovery_StandardResponseModes asserts the three IANA-registered
// response modes are advertised: query, fragment, form_post.
func TestDiscovery_StandardResponseModes(t *testing.T) {
	h := newDiscoveryTestProvider(t, &config.Config{JWTType: "RS256"})
	body := doDiscoveryRequest(t, h)
	modes := toStringSlice(t, body["response_modes_supported"])
	assert.Contains(t, modes, "query")
	assert.Contains(t, modes, "fragment")
	assert.Contains(t, modes, "form_post")
}
