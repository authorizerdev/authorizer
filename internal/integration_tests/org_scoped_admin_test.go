package integration_tests

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// genTestCertPEM returns a self-signed X.509 certificate in PEM form, valid
// input for CreateOrgSAMLConnection's idp_certificate (validateSAMLCertPEM
// parses it with x509.ParseCertificate).
func genTestCertPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-idp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// TestOrgScopedAdmin covers the Phase 1 org-scoped-admin authorization matrix:
// org-admins may manage their own org's SSO/SCIM/members, the confused-deputy
// (H2) hole is closed, the bare "admin" role is NOT accepted (H1), org
// create/delete stays super-admin-only, and the last-admin guard holds.
func TestOrgScopedAdmin(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Auth-mode switches operate on the shared gin request headers. requireOrgAdmin
	// reads the caller identity from this request (admin cookie or bearer token).
	asSuperAdmin := func() {
		clearCookies(ts)
		ts.GinContext.Request.Header.Del("Authorization")
		setAdminCookie(t, ts)
	}
	asUser := func(token string) {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+token)
	}
	asAnon := func() {
		clearCookies(ts)
		ts.GinContext.Request.Header.Del("Authorization")
	}

	// signupUser creates a fresh verified user and returns (id, accessToken).
	signupUser := func() (string, string) {
		asAnon()
		email := "org_admin_" + uuid.NewString() + "@authorizer.test"
		password := "Password@123"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, res.User)
		require.NotNil(t, res.AccessToken)
		require.NotEmpty(t, *res.AccessToken)
		return res.User.ID, *res.AccessToken
	}

	createOrg := func(prefix string) string {
		asSuperAdmin()
		org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{
			Name: prefix + "-" + uuid.NewString(),
		})
		require.NoError(t, err)
		return org.ID
	}

	addMember := func(orgID, userID string, roles []string) {
		asSuperAdmin()
		_, err := ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  orgID,
			UserID: userID,
			Roles:  roles,
		})
		require.NoError(t, err)
	}

	createOIDCConn := func(orgID string) string {
		asSuperAdmin()
		conn, err := ts.GraphQLProvider.CreateOrgOidcConnection(ctx, &model.CreateOrgOIDCConnectionRequest{
			OrgID:        orgID,
			Name:         "conn-" + uuid.NewString(),
			IssuerURL:    "https://idp-" + uuid.NewString() + ".example.com",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		})
		require.NoError(t, err)
		return conn.ID
	}

	t.Run("org-admin manages its own org's OIDC/SAML/SCIM", func(t *testing.T) {
		orgID := createOrg("acme")
		adminID, adminTok := signupUser()
		addMember(orgID, adminID, []string{constants.OrgRoleAdmin})

		asUser(adminTok)

		// OIDC: create, get, update, delete.
		oidc, err := ts.GraphQLProvider.CreateOrgOidcConnection(ctx, &model.CreateOrgOIDCConnectionRequest{
			OrgID:        orgID,
			Name:         "oidc-" + uuid.NewString(),
			IssuerURL:    "https://oidc-" + uuid.NewString() + ".example.com",
			ClientID:     "cid",
			ClientSecret: "secret",
		})
		require.NoError(t, err)
		require.NotNil(t, oidc)

		got, err := ts.GraphQLProvider.OrgOidcConnection(ctx, &model.OrgOIDCConnectionRequest{ID: refs.NewStringRef(oidc.ID)})
		require.NoError(t, err)
		require.Equal(t, orgID, got.OrgID)

		_, err = ts.GraphQLProvider.UpdateOrgOidcConnection(ctx, &model.UpdateOrgOIDCConnectionRequest{
			ID:   oidc.ID,
			Name: refs.NewStringRef("renamed"),
		})
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.DeleteOrgOidcConnection(ctx, &model.OrgOIDCConnectionRequest{ID: refs.NewStringRef(oidc.ID)})
		require.NoError(t, err)

		// SAML: create then delete.
		saml, err := ts.GraphQLProvider.CreateOrgSamlConnection(ctx, &model.CreateOrgSAMLConnectionRequest{
			OrgID:          orgID,
			Name:           "saml-" + uuid.NewString(),
			IdpEntityID:    "https://saml-" + uuid.NewString() + ".example.com",
			IdpSsoURL:      "https://saml-sso.example.com/sso",
			IdpCertificate: genTestCertPEM(t),
		})
		require.NoError(t, err)
		_, err = ts.GraphQLProvider.DeleteOrgSamlConnection(ctx, &model.OrgSAMLConnectionRequest{ID: refs.NewStringRef(saml.ID)})
		require.NoError(t, err)

		// SCIM: create, rotate, delete.
		scim, err := ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: orgID})
		require.NoError(t, err)
		require.NotEmpty(t, scim.Token)
		_, err = ts.GraphQLProvider.RotateScimToken(ctx, &model.ScimEndpointRequest{OrgID: orgID})
		require.NoError(t, err)
		_, err = ts.GraphQLProvider.DeleteScimEndpoint(ctx, &model.ScimEndpointRequest{OrgID: orgID})
		require.NoError(t, err)
	})

	t.Run("confused deputy: org-A admin cannot mutate org-B's connection", func(t *testing.T) {
		orgA := createOrg("orga")
		orgB := createOrg("orgb")
		adminID, adminTok := signupUser()
		addMember(orgA, adminID, []string{constants.OrgRoleAdmin})
		connB := createOIDCConn(orgB)

		asUser(adminTok)

		// Update by id only — auth must key on the loaded row's OrgID (orgB), deny.
		_, err := ts.GraphQLProvider.UpdateOrgOidcConnection(ctx, &model.UpdateOrgOIDCConnectionRequest{
			ID:   connB,
			Name: refs.NewStringRef("hijacked"),
		})
		require.Error(t, err)

		// Delete with {id: orgB-conn, org_id: orgA} — must not pass by claiming orgA.
		_, err = ts.GraphQLProvider.DeleteOrgOidcConnection(ctx, &model.OrgOIDCConnectionRequest{
			ID:    refs.NewStringRef(connB),
			OrgID: refs.NewStringRef(orgA),
		})
		require.Error(t, err)

		// Get with the same mismatched pair — deny.
		_, err = ts.GraphQLProvider.OrgOidcConnection(ctx, &model.OrgOIDCConnectionRequest{
			ID:    refs.NewStringRef(connB),
			OrgID: refs.NewStringRef(orgA),
		})
		require.Error(t, err)

		// The connection in orgB must still be intact (mutation did not land).
		asSuperAdmin()
		still, err := ts.GraphQLProvider.OrgOidcConnection(ctx, &model.OrgOIDCConnectionRequest{ID: refs.NewStringRef(connB)})
		require.NoError(t, err)
		require.Equal(t, orgB, still.OrgID)
	})

	t.Run("bare admin role is not accepted (H1)", func(t *testing.T) {
		orgID := createOrg("bare")
		userID, tok := signupUser()
		addMember(orgID, userID, []string{"admin", "billing"}) // app-level bare admin

		asUser(tok)
		_, err := ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: orgID})
		require.Error(t, err)
		_, err = ts.GraphQLProvider.CreateOrgOidcConnection(ctx, &model.CreateOrgOIDCConnectionRequest{
			OrgID:        orgID,
			Name:         "x",
			IssuerURL:    "https://x-" + uuid.NewString() + ".example.com",
			ClientID:     "c",
			ClientSecret: "s",
		})
		require.Error(t, err)
	})

	t.Run("plain member and non-member are denied", func(t *testing.T) {
		orgID := createOrg("plain")
		memberID, memberTok := signupUser()
		addMember(orgID, memberID, []string{"billing"})
		_, outsiderTok := signupUser()

		asUser(memberTok)
		_, err := ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: orgID})
		require.Error(t, err)

		asUser(outsiderTok)
		_, err = ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: orgID})
		require.Error(t, err)

		asAnon()
		_, err = ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: orgID})
		require.Error(t, err)
	})

	t.Run("super-admin retains full access (regression)", func(t *testing.T) {
		orgID := createOrg("super")
		asSuperAdmin()
		scim, err := ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: orgID})
		require.NoError(t, err)
		require.NotEmpty(t, scim.Token)
		conn := createOIDCConn(orgID) // helper runs as super-admin
		require.NotEmpty(t, conn)
	})

	t.Run("org-admin cannot create or delete the organization itself", func(t *testing.T) {
		orgID := createOrg("tenant")
		adminID, adminTok := signupUser()
		addMember(orgID, adminID, []string{constants.OrgRoleAdmin})

		asUser(adminTok)
		_, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{Name: "new-" + uuid.NewString()})
		require.Error(t, err)
		_, err = ts.GraphQLProvider.DeleteOrganization(ctx, &model.OrganizationRequest{ID: orgID})
		require.Error(t, err)
	})

	t.Run("last-admin guard", func(t *testing.T) {
		orgID := createOrg("lastadmin")
		admin1ID, admin1Tok := signupUser()
		addMember(orgID, admin1ID, []string{constants.OrgRoleAdmin})

		// Removing the only org_admin is refused for a non-super-admin caller.
		asUser(admin1Tok)
		_, err := ts.GraphQLProvider.RemoveOrgMember(ctx, &model.RemoveOrgMemberRequest{
			OrgID:  orgID,
			UserID: admin1ID,
		})
		require.Error(t, err)

		// Promote a second org_admin, then removing a non-last admin is allowed.
		admin2ID, _ := signupUser()
		asUser(admin1Tok)
		_, err = ts.GraphQLProvider.AddOrgMember(ctx, &model.AddOrgMemberRequest{
			OrgID:  orgID,
			UserID: admin2ID,
			Roles:  []string{constants.OrgRoleAdmin},
		})
		require.NoError(t, err, "org-admin may grant org_admin to another member of its own org")

		asUser(admin1Tok)
		_, err = ts.GraphQLProvider.RemoveOrgMember(ctx, &model.RemoveOrgMemberRequest{
			OrgID:  orgID,
			UserID: admin2ID,
		})
		require.NoError(t, err)

		// A super-admin is exempt: may remove the now-last admin.
		asSuperAdmin()
		_, err = ts.GraphQLProvider.RemoveOrgMember(ctx, &model.RemoveOrgMemberRequest{
			OrgID:  orgID,
			UserID: admin1ID,
		})
		require.NoError(t, err)
	})
}
