package http_handlers

import (
	"context"
	"encoding/base64"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/crewjam/saml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const (
	testIDPEntityID = "https://idp.example.com/saml/idp/acme/metadata"
	testIDPSSOURL   = "https://idp.example.com/saml/idp/acme/sso"
	testSPEntityID  = "https://sp.example.com/saml/metadata"
	testSPACSURL    = "https://sp.example.com/saml/acs"
)

// fakeSPStore satisfies samlSPRegistry.storage with a fixed set of SP records.
type fakeSPStore struct {
	byEntity map[string]*schemas.SAMLServiceProvider
}

func (f fakeSPStore) GetSAMLServiceProviderByOrgAndEntityID(_ context.Context, orgID, entityID string) (*schemas.SAMLServiceProvider, error) {
	sp := f.byEntity[entityID]
	if sp == nil || sp.OrgID != orgID {
		return nil, assert.AnError
	}
	return sp, nil
}

func mustURL(t *testing.T, raw string) url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return *u
}

func testUser() *schemas.User {
	return &schemas.User{
		ID:         "user-1",
		Email:      refs.NewStringRef("alice@example.com"),
		GivenName:  refs.NewStringRef("Alice"),
		FamilyName: refs.NewStringRef("Smith"),
	}
}

func testSPRecord() *schemas.SAMLServiceProvider {
	return &schemas.SAMLServiceProvider{
		ID:           "sp-1",
		OrgID:        "org-1",
		EntityID:     testSPEntityID,
		ACSURL:       testSPACSURL,
		NameIDFormat: defaultSAMLNameIDFormat,
		IsActive:     true,
	}
}

// buildTestIDP wires a crewjam IdentityProvider signed by a fresh keypair and
// backed by the given SP registry. Returns the IdP, its signing cert PEM, and the
// consuming crewjam ServiceProvider that trusts it.
func buildTestIDP(t *testing.T, store fakeSPStore) (*saml.IdentityProvider, *saml.ServiceProvider) {
	t.Helper()
	privPEM, certPEM, err := crypto.NewSAMLSigningKeypair("test idp")
	require.NoError(t, err)
	priv, err := crypto.ParseRsaPrivateKeyFromPemStr(privPEM)
	require.NoError(t, err)
	cert, err := crypto.ParseCertificateFromPemStr(certPEM)
	require.NoError(t, err)

	idp := &saml.IdentityProvider{
		Key:             priv,
		Certificate:     cert,
		MetadataURL:     mustURL(t, testIDPEntityID),
		SSOURL:          mustURL(t, testIDPSSOURL),
		SignatureMethod: rsaSHA256SigMethod,
		ServiceProviderProvider: &samlSPRegistry{
			storage: store,
			orgID:   "org-1",
			ctx:     context.Background(),
		},
	}

	consumingSP := &saml.ServiceProvider{
		EntityID:    testSPEntityID,
		MetadataURL: mustURL(t, testSPEntityID),
		AcsURL:      mustURL(t, testSPACSURL),
		IDPMetadata: buildIDPMetadata(testIDPEntityID, testIDPSSOURL, []*schemas.SAMLIDPKey{
			{CertPEM: certPEM, Status: schemas.SAMLIDPKeyStatusCurrent},
		}),
	}
	return idp, consumingSP
}

// samlResponseBytes builds the SAML response for an assertion-populated request
// and base64-decodes it (PostBinding is what WriteResponse serialises into the
// auto-POST form, minus the HTML layer).
func samlResponseBytes(t *testing.T, idpReq *saml.IdpAuthnRequest) []byte {
	t.Helper()
	form, err := idpReq.PostBinding()
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(form.SAMLResponse)
	require.NoError(t, err)
	return decoded
}

// TestSAMLIDPAssertionRoundTrip is the security-critical proof: an assertion this
// package emits must pass every check the SP side relies on (signature bound to
// the IdP cert, Audience/Recipient/Destination, and InResponseTo) — verified by
// feeding it back through a real crewjam ServiceProvider.ParseXMLResponse.
func TestSAMLIDPAssertionRoundTrip(t *testing.T) {
	spRec := testSPRecord()
	spRec.MappedAttributes = refs.NewStringRef(`{"email":"email","given_name":"firstName","family_name":"lastName"}`)
	store := fakeSPStore{byEntity: map[string]*schemas.SAMLServiceProvider{testSPEntityID: spRec}}
	idp, consumingSP := buildTestIDP(t, store)

	// The SP starts an SP-initiated flow.
	authnReq, err := consumingSP.MakeAuthenticationRequest(testIDPSSOURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	require.NoError(t, err)
	redirectURL, err := authnReq.Redirect("relay-123", consumingSP)
	require.NoError(t, err)

	// The IdP parses and validates it (this binds the SP + ACS URL).
	httpReq := httptest.NewRequest("GET", "/saml/idp/acme/sso?"+redirectURL.RawQuery, nil)
	idpReq, err := saml.NewIdpAuthnRequest(idp, httpReq)
	require.NoError(t, err)
	require.NoError(t, idpReq.Validate(), "a registered SP with a matching ACS must validate")

	// Emit the assertion with mapped attributes.
	session := buildSAMLSession(testUser(), spRec)
	maker := mappedAssertionMaker{attributes: buildMappedAttributes(testUser(), spRec)}
	require.NoError(t, maker.MakeAssertion(idpReq, session))
	// The SP consumes it — the assertion must satisfy signature + all bindings.
	decoded := samlResponseBytes(t, idpReq)
	assertion, err := consumingSP.ParseXMLResponse(decoded, []string{authnReq.ID}, consumingSP.AcsURL)
	require.NoError(t, err, "the emitted assertion must pass the SP's XSW/signature/audience checks")

	require.NotNil(t, assertion.Subject)
	require.NotNil(t, assertion.Subject.NameID)
	assert.Equal(t, "alice@example.com", assertion.Subject.NameID.Value)

	got := map[string]string{}
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if len(attr.Values) > 0 {
				got[attr.Name] = attr.Values[0].Value
			}
		}
	}
	assert.Equal(t, "alice@example.com", got["email"])
	assert.Equal(t, "Alice", got["firstName"])
	assert.Equal(t, "Smith", got["lastName"])
}

// TestSAMLIDPAudienceIsolation proves an assertion minted for SP-A does not
// validate at SP-B (Audience is bound to the requesting SP's EntityID).
func TestSAMLIDPAudienceIsolation(t *testing.T) {
	spRec := testSPRecord()
	store := fakeSPStore{byEntity: map[string]*schemas.SAMLServiceProvider{testSPEntityID: spRec}}
	idp, consumingSP := buildTestIDP(t, store)

	authnReq, err := consumingSP.MakeAuthenticationRequest(testIDPSSOURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	require.NoError(t, err)
	redirectURL, err := authnReq.Redirect("", consumingSP)
	require.NoError(t, err)
	idpReq, err := saml.NewIdpAuthnRequest(idp, httptest.NewRequest("GET", "/sso?"+redirectURL.RawQuery, nil))
	require.NoError(t, err)
	require.NoError(t, idpReq.Validate())
	session := buildSAMLSession(testUser(), spRec)
	require.NoError(t, mappedAssertionMaker{attributes: buildMappedAttributes(testUser(), spRec)}.MakeAssertion(idpReq, session))
	decoded := samlResponseBytes(t, idpReq)

	// A different SP (different EntityID/ACS) must reject the assertion.
	otherSP := &saml.ServiceProvider{
		EntityID:    "https://evil.example.com/saml/metadata",
		MetadataURL: mustURL(t, "https://evil.example.com/saml/metadata"),
		AcsURL:      mustURL(t, "https://evil.example.com/saml/acs"),
		IDPMetadata: consumingSP.IDPMetadata,
	}
	_, err = otherSP.ParseXMLResponse(decoded, []string{authnReq.ID}, otherSP.AcsURL)
	assert.Error(t, err, "an assertion for SP-A must not validate at SP-B")
}

// TestSAMLIDPUnknownServiceProviderRejected proves an AuthnRequest from an
// unregistered SP is refused at Validate (the strict-binding guard).
func TestSAMLIDPUnknownServiceProviderRejected(t *testing.T) {
	store := fakeSPStore{byEntity: map[string]*schemas.SAMLServiceProvider{}} // registry is empty
	idp, consumingSP := buildTestIDP(t, store)
	authnReq, err := consumingSP.MakeAuthenticationRequest(testIDPSSOURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	require.NoError(t, err)
	redirectURL, err := authnReq.Redirect("", consumingSP)
	require.NoError(t, err)
	idpReq, err := saml.NewIdpAuthnRequest(idp, httptest.NewRequest("GET", "/sso?"+redirectURL.RawQuery, nil))
	require.NoError(t, err)
	assert.Error(t, idpReq.Validate(), "an AuthnRequest from an unregistered SP must be rejected")
}

func TestBuildMappedAttributes(t *testing.T) {
	user := testUser()

	// Defaults (samlDefaultAttributeMapping) when no override is set.
	def := buildMappedAttributes(user, testSPRecord())
	defNames := map[string]string{}
	for _, a := range def {
		defNames[a.Name] = a.Values[0].Value
	}
	assert.Equal(t, "alice@example.com", defNames["email"])
	assert.Equal(t, "Alice", defNames["firstName"])

	// Custom mapping emits exactly the configured attribute names.
	sp := testSPRecord()
	sp.MappedAttributes = refs.NewStringRef(`{"email":"urn:mail","given_name":"gn"}`)
	custom := buildMappedAttributes(user, sp)
	names := map[string]string{}
	for _, a := range custom {
		names[a.Name] = a.Values[0].Value
	}
	assert.Equal(t, "alice@example.com", names["urn:mail"])
	assert.Equal(t, "Alice", names["gn"])
	_, hasDefault := names["email"]
	assert.False(t, hasDefault, "custom mapping must not emit default attribute names")

	// Empty profile fields are skipped.
	empty := buildMappedAttributes(&schemas.User{ID: "u2"}, testSPRecord())
	assert.Empty(t, empty, "no attributes when the profile has no mapped values")
}

func TestBuildSAMLSessionNameID(t *testing.T) {
	user := testUser()

	// emailAddress format → email as NameID.
	sp := testSPRecord()
	sess := buildSAMLSession(user, sp)
	assert.Equal(t, "alice@example.com", sess.NameID)
	assert.Equal(t, defaultSAMLNameIDFormat, sess.NameIDFormat)

	// persistent format → stable user id as NameID.
	sp.NameIDFormat = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
	sess = buildSAMLSession(user, sp)
	assert.Equal(t, "user-1", sess.NameID)
}

func TestPublishedSAMLKeys(t *testing.T) {
	keys := []*schemas.SAMLIDPKey{
		{ID: "a", Status: schemas.SAMLIDPKeyStatusCurrent},
		{ID: "b", Status: schemas.SAMLIDPKeyStatusActive},
		{ID: "c", Status: schemas.SAMLIDPKeyStatusRetired},
	}
	pub := publishedSAMLKeys(keys)
	ids := []string{}
	for _, k := range pub {
		ids = append(ids, k.ID)
	}
	assert.ElementsMatch(t, []string{"a", "b"}, ids, "retired keys must not be published")
}
