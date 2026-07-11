package integration_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestOrgHomeRealmDiscovery covers Phase 3: the public /api/v1/org-discovery
// endpoint that routes a login email's verified domain to the owning org's SSO
// login URL. It asserts the minimal, privacy-preserving contract (review M5):
// SAML precedence, no leak of org id/name/slug beyond login_url, no-match and
// no-connection indistinguishable, normalization parity with Phase-2 storage,
// the flag gate, and per-IP rate limiting. DNS is never touched (verified rows
// are inserted directly).
func TestOrgHomeRealmDiscovery(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableOrgDiscovery = true
	// burst=1 / rps=1 makes the rate-limit trip deterministic: a second request
	// from the SAME ip within the test's sub-second runtime is denied. Every
	// functional case below uses a UNIQUE source ip, so each gets its own bucket
	// with a fresh token and never trips.
	cfg.RateLimitRPS = 1
	cfg.RateLimitBurst = 1
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	router := gin.New()
	router.GET("/api/v1/org-discovery", ts.HttpProvider.OrgDiscoveryHandler())

	type connection struct {
		Type     string `json:"type"`
		LoginURL string `json:"login_url"`
	}
	type discoveryResp struct {
		Connection *connection `json:"connection"`
		// Guard fields: these MUST never appear in the response (privacy M5).
		OrganizationID   *string `json:"organization_id"`
		OrganizationName *string `json:"organization_name"`
		OrgID            *string `json:"org_id"`
	}
	call := func(email, ip string) (int, string, discoveryResp) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/org-discovery?email="+url.QueryEscape(email), nil)
		req.RemoteAddr = ip + ":40000"
		router.ServeHTTP(w, req)
		var out discoveryResp
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, w.Body.String(), out
	}

	asSuperAdmin := func() {
		clearCookies(ts)
		ts.GinContext.Request.Header.Del("Authorization")
		setAdminCookie(t, ts)
	}
	createOrg := func(prefix string) *model.Organization {
		asSuperAdmin()
		org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: prefix + "-" + strings.ToLower(uuid.NewString()[:8]),
		})
		require.NoError(t, err)
		return org
	}
	addVerifiedDomain := func(orgID, prefix string) string {
		domain := prefix + "-" + strings.ToLower(uuid.NewString()[:8]) + ".com"
		now := time.Now().Unix()
		_, err := ts.StorageProvider.AddOrgDomain(ctx, &schemas.OrgDomain{
			ID: domain, OrgID: orgID, Domain: domain, VerifiedAt: now, CreatedAt: now, UpdatedAt: now,
		})
		require.NoError(t, err)
		return domain
	}
	addConnection := func(orgID, kind string) {
		_, err := ts.StorageProvider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
			OrgID:     orgID,
			Kind:      kind,
			IsActive:  true,
			IssuerURL: "https://idp-" + uuid.NewString() + ".example.com",
		})
		require.NoError(t, err)
	}

	t.Run("verified SAML domain returns saml login_url", func(t *testing.T) {
		org := createOrg("saml")
		domain := addVerifiedDomain(org.ID, "saml")
		addConnection(org.ID, constants.TrustKindSSOSAML)

		code, body, resp := call("alice@"+domain, "203.0.113.1")
		require.Equal(t, http.StatusOK, code)
		require.NotNil(t, resp.Connection)
		require.Equal(t, "saml", resp.Connection.Type)
		require.Equal(t, "/oauth/saml/"+org.Name+"/login", resp.Connection.LoginURL)
		// Privacy: nothing beyond login_url leaks the tenant identity.
		require.Nil(t, resp.OrganizationID)
		require.Nil(t, resp.OrganizationName)
		require.Nil(t, resp.OrgID)
		require.NotContains(t, body, org.ID)
	})

	t.Run("verified OIDC domain returns oidc login_url", func(t *testing.T) {
		org := createOrg("oidc")
		domain := addVerifiedDomain(org.ID, "oidc")
		addConnection(org.ID, constants.TrustKindSSOOIDC)

		code, _, resp := call("bob@"+domain, "203.0.113.2")
		require.Equal(t, http.StatusOK, code)
		require.NotNil(t, resp.Connection)
		require.Equal(t, "oidc", resp.Connection.Type)
		require.Equal(t, "/oauth/sso/"+org.Name+"/login", resp.Connection.LoginURL)
	})

	t.Run("SAML takes precedence when both connections exist", func(t *testing.T) {
		org := createOrg("both")
		domain := addVerifiedDomain(org.ID, "both")
		addConnection(org.ID, constants.TrustKindSSOOIDC)
		addConnection(org.ID, constants.TrustKindSSOSAML)

		code, _, resp := call("carol@"+domain, "203.0.113.3")
		require.Equal(t, http.StatusOK, code)
		require.NotNil(t, resp.Connection)
		require.Equal(t, "saml", resp.Connection.Type)
	})

	t.Run("verified domain with no connection returns null", func(t *testing.T) {
		org := createOrg("noconn")
		domain := addVerifiedDomain(org.ID, "noconn")

		code, _, resp := call("dave@"+domain, "203.0.113.4")
		require.Equal(t, http.StatusOK, code)
		require.Nil(t, resp.Connection)
	})

	t.Run("unknown domain returns null (indistinguishable from no-connection)", func(t *testing.T) {
		code, _, resp := call("eve@unknown-"+uuid.NewString()[:8]+".com", "203.0.113.5")
		require.Equal(t, http.StatusOK, code)
		require.Nil(t, resp.Connection)
	})

	t.Run("normalization parity: mixed-case email resolves to stored domain", func(t *testing.T) {
		org := createOrg("norm")
		domain := addVerifiedDomain(org.ID, "norm") // stored lowercase
		addConnection(org.ID, constants.TrustKindSSOSAML)

		// Upper-cased local + domain and surrounding whitespace must normalize to
		// the exact value Phase-2 stored (same normalizeDomain).
		code, _, resp := call("  FRANK@"+strings.ToUpper(domain)+"  ", "203.0.113.6")
		require.Equal(t, http.StatusOK, code)
		require.NotNil(t, resp.Connection)
		require.Equal(t, "/oauth/saml/"+org.Name+"/login", resp.Connection.LoginURL)
	})

	t.Run("malformed email returns 400", func(t *testing.T) {
		// Unique IP per case: the rate limiter runs before parsing, so reusing one
		// IP would turn later requests into 429s instead of the 400 under test.
		for i, bad := range []string{"", "not-an-email", "@no-local.com", "user@", "user@localhost", "user@bad domain.com"} {
			code, _, _ := call(bad, fmt.Sprintf("203.0.113.%d", 70+i))
			require.Equal(t, http.StatusBadRequest, code, "email %q should be rejected", bad)
		}
	})

	t.Run("flag disabled returns 404", func(t *testing.T) {
		ts.Config.EnableOrgDiscovery = false
		defer func() { ts.Config.EnableOrgDiscovery = true }()

		org := createOrg("disabled")
		domain := addVerifiedDomain(org.ID, "disabled")
		addConnection(org.ID, constants.TrustKindSSOSAML)

		code, _, resp := call("grace@"+domain, "203.0.113.8")
		require.Equal(t, http.StatusNotFound, code)
		require.Nil(t, resp.Connection)
	})

	t.Run("rate limit trips per IP", func(t *testing.T) {
		org := createOrg("rl")
		domain := addVerifiedDomain(org.ID, "rl")
		addConnection(org.ID, constants.TrustKindSSOSAML)

		ip := "198.51.100.9"
		// First request from this IP consumes the single burst token.
		code, _, _ := call("henry@"+domain, ip)
		require.Equal(t, http.StatusOK, code)
		// Second request from the SAME IP, within the same instant, is denied.
		code, _, _ = call("henry@"+domain, ip)
		require.Equal(t, http.StatusTooManyRequests, code)
	})
}
