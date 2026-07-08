package http_handlers

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// SAML test fixtures. The org-A connection is the "victim" SP; org-B is a
// separate SP used to prove cross-org rejection.
const (
	samlTestHost       = "https://auth.example.com"
	samlOrgASlug       = "org-a"
	samlOrgAID         = "org-a-id"
	samlIdPEntityID    = "https://idp.example.com/saml/metadata"
	samlIdPSSOURL      = "https://idp.example.com/sso"
	samlTestNameID     = "corp-user-42"
	samlTestEmail      = "user@corp.example.com"
	samlTestRequestID  = "id-authnrequest-abc123"
	samlTimeFmt        = "2006-01-02T15:04:05Z07:00"
	samlOrgBSPEntityID = "https://auth.example.com/oauth/saml/org-b/metadata"
	samlOrgBACSURL     = "https://auth.example.com/oauth/saml/org-b/acs"
)

// samlNow is the fixed "current" time the tests pin saml.TimeNow to.
var samlNow = time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)

// samlIdP is a generated test IdP signing keypair + self-signed certificate.
type samlIdP struct {
	key     *rsa.PrivateKey
	certDER []byte
	certPEM string
}

func newSAMLIdP(t *testing.T) *samlIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-idp"},
		// Wide validity window: goxmldsig checks the cert against the real wall
		// clock (not saml.TimeNow), so this must bracket the actual test run time.
		NotBefore: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return &samlIdP{key: key, certDER: der, certPEM: pemStr}
}

// assertionParams control how a test assertion is built and (optionally) tampered.
type assertionParams struct {
	assertionID  string
	issuer       string
	nameID       string
	audience     string
	recipient    string
	destination  string
	inResponseTo string
	notBefore    time.Time
	notOnOrAfter time.Time
	email        string
	// attributes, when non-nil, replaces the default single "email" attribute.
	attributes map[string]string
	sign       bool
	// tamperAfterSign flips a byte in the signed NameID to simulate a broken /
	// wrapped signature (the consumed element differs from what was signed).
	tamperAfterSign bool
}

func defaultAssertionParams() assertionParams {
	return assertionParams{
		assertionID:  "id-assertion-0001",
		issuer:       samlIdPEntityID,
		nameID:       samlTestNameID,
		audience:     samlTestHost + "/oauth/saml/" + samlOrgASlug + "/metadata",
		recipient:    samlTestHost + "/oauth/saml/" + samlOrgASlug + "/acs",
		destination:  samlTestHost + "/oauth/saml/" + samlOrgASlug + "/acs",
		inResponseTo: samlTestRequestID,
		notBefore:    samlNow.Add(-2 * time.Minute),
		notOnOrAfter: samlNow.Add(5 * time.Minute),
		email:        samlTestEmail,
		sign:         true,
	}
}

// buildAssertionElement builds the (unsigned) <saml:Assertion> element using the
// crewjam struct Element() methods so timestamps/namespaces are well-formed.
func buildAssertionElement(p assertionParams) *etree.Element {
	a := saml.Assertion{
		ID:           p.assertionID,
		IssueInstant: samlNow,
		Version:      "2.0",
		Issuer:       saml.Issuer{Value: p.issuer},
		Subject: &saml.Subject{
			NameID: &saml.NameID{Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent", Value: p.nameID},
			SubjectConfirmations: []saml.SubjectConfirmation{{
				Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
				SubjectConfirmationData: &saml.SubjectConfirmationData{
					InResponseTo: p.inResponseTo,
					Recipient:    p.recipient,
					NotOnOrAfter: p.notOnOrAfter,
				},
			}},
		},
		Conditions: &saml.Conditions{
			NotBefore:            p.notBefore,
			NotOnOrAfter:         p.notOnOrAfter,
			AudienceRestrictions: []saml.AudienceRestriction{{Audience: saml.Audience{Value: p.audience}}},
		},
		AuthnStatements: []saml.AuthnStatement{{AuthnInstant: samlNow}},
	}
	attrs := p.attributes
	if attrs == nil && p.email != "" {
		attrs = map[string]string{"email": p.email}
	}
	if len(attrs) > 0 {
		stmt := saml.AttributeStatement{}
		for name, val := range attrs {
			stmt.Attributes = append(stmt.Attributes, saml.Attribute{
				Name:   name,
				Values: []saml.AttributeValue{{Type: "xs:string", Value: val}},
			})
		}
		a.AttributeStatements = []saml.AttributeStatement{stmt}
	}
	return a.Element()
}

// parseValidAssertion builds, signs, and validates a response, returning the
// consumed assertion (fails the test if validation does not succeed).
func parseValidAssertion(t *testing.T, idp *samlIdP, p assertionParams) *saml.Assertion {
	t.Helper()
	raw := makeSAMLResponse(t, idp, p)
	a, err := parseWith(t, samlTestConn(idp), raw, []string{p.inResponseTo})
	require.NoError(t, err)
	return a
}

// makeSAMLResponse builds a base64-encoded <samlp:Response> wrapping the (optionally
// signed) assertion, ready to POST to the ACS.
func makeSAMLResponse(t *testing.T, idp *samlIdP, p assertionParams) string {
	t.Helper()
	assertionEl := buildAssertionElement(p)

	if p.sign {
		sctx, err := dsig.NewSigningContext(idp.key, [][]byte{idp.certDER})
		require.NoError(t, err)
		sctx.Hash = crypto.SHA256
		sctx.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
		signed, err := sctx.SignEnveloped(assertionEl)
		require.NoError(t, err)
		assertionEl = signed
	}
	if p.tamperAfterSign {
		// Mutate the signed subtree AFTER signing: the digest no longer matches the
		// consumed element (the essence of XML Signature Wrapping / integrity breaks).
		if nameID := assertionEl.FindElement("//NameID"); nameID != nil {
			nameID.SetText("attacker-substituted-subject")
		}
	}

	respEl := etree.NewElement("samlp:Response")
	respEl.CreateAttr("xmlns:samlp", "urn:oasis:names:tc:SAML:2.0:protocol")
	respEl.CreateAttr("xmlns:saml", "urn:oasis:names:tc:SAML:2.0:assertion")
	respEl.CreateAttr("ID", "id-response-0001")
	respEl.CreateAttr("Version", "2.0")
	respEl.CreateAttr("IssueInstant", samlNow.Format(samlTimeFmt))
	if p.inResponseTo != "" {
		respEl.CreateAttr("InResponseTo", p.inResponseTo)
	}
	if p.destination != "" {
		respEl.CreateAttr("Destination", p.destination)
	}
	respEl.CreateElement("saml:Issuer").SetText(p.issuer)
	status := respEl.CreateElement("samlp:Status")
	status.CreateElement("samlp:StatusCode").CreateAttr("Value", saml.StatusSuccess)
	respEl.AddChild(assertionEl)

	doc := etree.NewDocument()
	doc.SetRoot(respEl)
	xmlBytes, err := doc.WriteToBytes()
	require.NoError(t, err)
	return string(xmlBytes)
}

// samlTestConn builds an sso_saml TrustedIssuer row for org A trusting idp.
func samlTestConn(idp *samlIdP) *schemas.TrustedIssuer {
	return &schemas.TrustedIssuer{
		Kind:           "sso_saml",
		OrgID:          samlOrgAID,
		IssuerURL:      samlIdPEntityID,
		SAMLSSOURL:     refs.NewStringRef(samlIdPSSOURL),
		SAMLIDPCertPEM: refs.NewStringRef(idp.certPEM),
		IsActive:       true,
	}
}

// parseWith runs buildSAMLServiceProvider + ParseXMLResponse under pinned time.
func parseWith(t *testing.T, conn *schemas.TrustedIssuer, rawXML string, requestIDs []string) (*saml.Assertion, error) {
	t.Helper()
	restore := saml.TimeNow
	saml.TimeNow = func() time.Time { return samlNow }
	defer func() { saml.TimeNow = restore }()

	sp, err := buildSAMLServiceProvider(conn, samlTestHost, samlOrgASlug)
	require.NoError(t, err)
	return sp.ParseXMLResponse([]byte(rawXML), requestIDs, sp.AcsURL)
}

func TestSAML_ValidSignedAssertionAccepted(t *testing.T) {
	idp := newSAMLIdP(t)
	raw := makeSAMLResponse(t, idp, defaultAssertionParams())
	assertion, err := parseWith(t, samlTestConn(idp), raw, []string{samlTestRequestID})
	require.NoError(t, err)
	require.NotNil(t, assertion.Subject)
	require.NotNil(t, assertion.Subject.NameID)
	assert.Equal(t, samlTestNameID, assertion.Subject.NameID.Value)
	assert.Equal(t, samlTestEmail, samlAttr(assertion, "email"))
}

func TestSAML_UnsignedAssertionRejected(t *testing.T) {
	idp := newSAMLIdP(t)
	p := defaultAssertionParams()
	p.sign = false
	raw := makeSAMLResponse(t, idp, p)
	_, err := parseWith(t, samlTestConn(idp), raw, []string{samlTestRequestID})
	require.Error(t, err)
}

func TestSAML_TamperedSignatureRejected_XSW(t *testing.T) {
	idp := newSAMLIdP(t)
	p := defaultAssertionParams()
	p.tamperAfterSign = true
	raw := makeSAMLResponse(t, idp, p)
	_, err := parseWith(t, samlTestConn(idp), raw, []string{samlTestRequestID})
	require.Error(t, err)
}

func TestSAML_WrongSigningKeyRejected(t *testing.T) {
	idp := newSAMLIdP(t)
	attacker := newSAMLIdP(t) // signs with a different key than the conn trusts
	raw := makeSAMLResponse(t, attacker, defaultAssertionParams())
	_, err := parseWith(t, samlTestConn(idp), raw, []string{samlTestRequestID})
	require.Error(t, err)
}

// Cross-org: an assertion whose Audience/Recipient target Org B, presented at
// Org A's ACS, must be rejected (per-org audience + recipient binding).
func TestSAML_CrossOrgAudienceRejected(t *testing.T) {
	idp := newSAMLIdP(t)
	p := defaultAssertionParams()
	p.audience = samlOrgBSPEntityID
	p.recipient = samlOrgBACSURL
	p.destination = samlOrgBACSURL
	raw := makeSAMLResponse(t, idp, p)
	_, err := parseWith(t, samlTestConn(idp), raw, []string{samlTestRequestID})
	require.Error(t, err)
}

func TestSAML_WrongRecipientRejected(t *testing.T) {
	idp := newSAMLIdP(t)
	p := defaultAssertionParams()
	p.recipient = "https://auth.example.com/oauth/saml/other/acs"
	raw := makeSAMLResponse(t, idp, p)
	_, err := parseWith(t, samlTestConn(idp), raw, []string{samlTestRequestID})
	require.Error(t, err)
}

func TestSAML_ExpiredAssertionRejected(t *testing.T) {
	idp := newSAMLIdP(t)
	p := defaultAssertionParams()
	p.notBefore = samlNow.Add(-2 * time.Hour)
	p.notOnOrAfter = samlNow.Add(-1 * time.Hour) // well beyond MaxClockSkew
	raw := makeSAMLResponse(t, idp, p)
	_, err := parseWith(t, samlTestConn(idp), raw, []string{samlTestRequestID})
	require.Error(t, err)
}

func TestSAML_InResponseToMismatchRejected(t *testing.T) {
	idp := newSAMLIdP(t)
	p := defaultAssertionParams()
	p.inResponseTo = "id-some-other-request"
	raw := makeSAMLResponse(t, idp, p)
	_, err := parseWith(t, samlTestConn(idp), raw, []string{samlTestRequestID})
	require.Error(t, err)
}

// IdP-initiated (no pending AuthnRequest) is rejected when the connection does
// not opt in: with an empty possibleRequestIDs and AllowIDPInitiated=false the
// library cannot bind the response to any request.
func TestSAML_IdPInitiatedRejectedWhenFlagOff(t *testing.T) {
	idp := newSAMLIdP(t)
	p := defaultAssertionParams()
	p.inResponseTo = "" // IdP-initiated: no InResponseTo
	raw := makeSAMLResponse(t, idp, p)
	conn := samlTestConn(idp)
	conn.SAMLAllowIDPInitiated = false
	_, err := parseWith(t, conn, raw, nil)
	require.Error(t, err)
}
