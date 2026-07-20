package integration_tests

import (
	"context"
	"encoding/base64"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// samlIDPTestHost is pinned via the X-Authorizer-URL header on every request so
// the IdP's derived entity ID / SSO URL are deterministic and match the
// consuming SP's AuthnRequest Destination.
const samlIDPTestHost = "http://localhost:8080"

// TestSAMLIDPEndToEnd drives the real IdP gin handlers end-to-end against SQLite
// storage, a real token/session, and a real crewjam ServiceProvider consuming the
// emitted assertion. It exercises: metadata, SP-initiated issuance for a member,
// the unauthenticated login bounce, the C1 non-member denial, and IdP-initiated
// gating.
func TestSAMLIDPEndToEnd(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	ctx := context.Background()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/saml/idp/:org_slug/metadata", ts.HttpProvider.SAMLIDPMetadataHandler())
	router.GET("/saml/idp/:org_slug/sso", ts.HttpProvider.SAMLIDPSSOHandler())
	router.POST("/saml/idp/:org_slug/sso", ts.HttpProvider.SAMLIDPSSOHandler())
	router.GET("/saml/idp/:org_slug/sso/:sp_id", ts.HttpProvider.SAMLIDPInitiatedHandler())

	slug := "acme-idp-" + uuid.NewString()[:8]
	org, err := ts.StorageProvider.AddOrganization(ctx, &schemas.Organization{Name: slug, Enabled: true})
	require.NoError(t, err)

	now := time.Now().Unix()
	memberEmail := "member-" + uuid.NewString()[:8] + "@acme.example.com"
	member, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef(memberEmail),
		EmailVerifiedAt: &now,
		GivenName:       refs.NewStringRef("Mem"),
		FamilyName:      refs.NewStringRef("Ber"),
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)
	_, err = ts.StorageProvider.AddOrgMembership(ctx, &schemas.OrgMembership{OrgID: org.ID, UserID: member.ID, Roles: "member"})
	require.NoError(t, err)

	// A verified user who is NOT a member of the org (the C1 attacker).
	outsider, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef("outsider-" + uuid.NewString()[:8] + "@evil.example.com"),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	spEntityID := "https://sp.example.test/metadata"
	spACS := "https://sp.example.test/acs"
	sp, err := ts.StorageProvider.AddSAMLServiceProvider(ctx, &schemas.SAMLServiceProvider{
		OrgID:        org.ID,
		Name:         "TestSP",
		EntityID:     spEntityID,
		ACSURL:       spACS,
		NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		IsActive:     true,
	})
	require.NoError(t, err)

	idpEntityID := samlIDPTestHost + "/saml/idp/" + slug + "/metadata"
	idpSSOURL := samlIDPTestHost + "/saml/idp/" + slug + "/sso"

	// ---- 1. Metadata ----
	t.Run("metadata publishes the signing cert", func(t *testing.T) {
		w := samlIDPGet(router, "/saml/idp/"+slug+"/metadata", "")
		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "X509Certificate")
		assert.Contains(t, w.Body.String(), idpEntityID)
	})

	// Consuming SP trusts the IdP by parsing the metadata we just served.
	metaXML := samlIDPGet(router, "/saml/idp/"+slug+"/metadata", "").Body.Bytes()
	idpMeta, err := samlsp.ParseMetadata(metaXML)
	require.NoError(t, err)

	newConsumingSP := func(allowIDPInitiated bool) *saml.ServiceProvider {
		return &saml.ServiceProvider{
			EntityID:          spEntityID,
			MetadataURL:       mustURL(t, spEntityID),
			AcsURL:            mustURL(t, spACS),
			IDPMetadata:       idpMeta,
			AllowIDPInitiated: allowIDPInitiated,
		}
	}

	// ---- 2. SP-initiated, member session → valid signed assertion ----
	t.Run("member receives a valid signed assertion", func(t *testing.T) {
		consumingSP := newConsumingSP(false)
		authnReq, err := consumingSP.MakeAuthenticationRequest(idpSSOURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
		require.NoError(t, err)
		redirect, err := authnReq.Redirect("relay-e2e", consumingSP)
		require.NoError(t, err)

		w := samlIDPGet(router, "/saml/idp/"+slug+"/sso?"+redirect.RawQuery, samlIDPLoginCookie(t, ts, member))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		decoded := extractSAMLResponse(t, w.Body.String())
		assertion, err := consumingSP.ParseXMLResponse(decoded, []string{authnReq.ID}, consumingSP.AcsURL)
		require.NoError(t, err, "the emitted assertion must satisfy the SP's signature/audience/InResponseTo checks")
		require.NotNil(t, assertion.Subject)
		require.NotNil(t, assertion.Subject.NameID)
		assert.Equal(t, memberEmail, assertion.Subject.NameID.Value)
	})

	// ---- 3. Unauthenticated → login bounce ----
	t.Run("unauthenticated request bounces to the login UI", func(t *testing.T) {
		consumingSP := newConsumingSP(false)
		authnReq, err := consumingSP.MakeAuthenticationRequest(idpSSOURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
		require.NoError(t, err)
		redirect, err := authnReq.Redirect("", consumingSP)
		require.NoError(t, err)

		w := samlIDPGet(router, "/saml/idp/"+slug+"/sso?"+redirect.RawQuery, "")
		require.Equal(t, http.StatusFound, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/app?redirect_uri=")
	})

	// ---- 4. C1: non-member session → issuance denied ----
	t.Run("non-member is denied issuance (C1)", func(t *testing.T) {
		consumingSP := newConsumingSP(false)
		authnReq, err := consumingSP.MakeAuthenticationRequest(idpSSOURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
		require.NoError(t, err)
		redirect, err := authnReq.Redirect("", consumingSP)
		require.NoError(t, err)

		w := samlIDPGet(router, "/saml/idp/"+slug+"/sso?"+redirect.RawQuery, samlIDPLoginCookie(t, ts, outsider))
		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "forbidden")
	})

	// ---- 5. IdP-initiated gating ----
	t.Run("idp-initiated is refused unless the SP opts in", func(t *testing.T) {
		w := samlIDPGet(router, "/saml/idp/"+slug+"/sso/"+sp.ID, samlIDPLoginCookie(t, ts, member))
		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "idp_initiated_disabled")
	})

	t.Run("idp-initiated issues an unsolicited assertion when enabled", func(t *testing.T) {
		sp.AllowIDPInitiated = true
		_, err := ts.StorageProvider.UpdateSAMLServiceProvider(ctx, sp)
		require.NoError(t, err)

		w := samlIDPGet(router, "/saml/idp/"+slug+"/sso/"+sp.ID, samlIDPLoginCookie(t, ts, member))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		consumingSP := newConsumingSP(true) // accept an unsolicited (no-InResponseTo) assertion
		decoded := extractSAMLResponse(t, w.Body.String())
		assertion, err := consumingSP.ParseXMLResponse(decoded, []string{}, consumingSP.AcsURL)
		require.NoError(t, err)
		assert.Equal(t, memberEmail, assertion.Subject.NameID.Value)
	})
}

// samlIDPGet issues a GET through the router with the IdP host pinned and an
// optional session cookie.
func samlIDPGet(router *gin.Engine, path, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Authorizer-URL", samlIDPTestHost)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// samlIDPLoginCookie mints a real browser session for the user (mirroring the
// login flow: CreateAuthToken + memory-store session) and returns the session
// cookie header value the IdP handler reads via cookie.GetSession.
func samlIDPLoginCookie(t *testing.T, ts *testSetup, user *schemas.User) string {
	t.Helper()
	authToken, err := ts.TokenProvider.CreateAuthToken(nil, &token.AuthTokenConfig{
		User:        user,
		Roles:       []string{"user"},
		Scope:       []string{"openid", "email"},
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		HostName:    samlIDPTestHost,
	})
	require.NoError(t, err)
	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionKey,
		constants.TokenTypeSessionToken+"_"+authToken.FingerPrint,
		authToken.FingerPrintHash,
		authToken.SessionTokenExpiresAt,
	))
	return constants.AppCookieName + "_session=" + url.PathEscape(authToken.FingerPrintHash)
}

// extractSAMLResponse pulls the base64 SAMLResponse out of the IdP's auto-POST
// HTML form. html/template escapes attribute values, so unescape before decoding.
func extractSAMLResponse(t *testing.T, body string) []byte {
	t.Helper()
	m := regexp.MustCompile(`name="SAMLResponse" value="([^"]*)"`).FindStringSubmatch(body)
	require.Len(t, m, 2, "SAMLResponse form field not found: %s", body)
	decoded, err := base64.StdEncoding.DecodeString(html.UnescapeString(m[1]))
	require.NoError(t, err)
	return decoded
}

func mustURL(t *testing.T, raw string) url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return *u
}
