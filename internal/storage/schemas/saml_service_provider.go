package schemas

import (
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// SAMLServiceProvider registers a downstream SAML 2.0 Service Provider (SP) that
// an Organization's users authenticate INTO, with Authorizer acting as the SAML
// Identity Provider (IdP). This is the architectural inverse of the sso_saml
// TrustedIssuer row (which registers an UPSTREAM IdP that Authorizer, as an SP,
// consumes). Do NOT conflate the two: a TrustedIssuer means "an issuer we trust";
// a SAMLServiceProvider means "an SP we issue signed assertions to".
//
// SECURITY INVARIANTS (enforced by the IdP handlers, not here):
//
//	I1 — EntityID and ACSURL are the ONLY trusted values. An incoming
//	     AuthnRequest's Issuer selects this record by (OrgID, EntityID); the
//	     assertion's Audience is set to this EntityID and the Recipient/Destination
//	     to this ACSURL. A request-supplied AssertionConsumerServiceURL that does
//	     not match this ACSURL is rejected (crewjam getACSEndpoint), which is the
//	     open-redirect / assertion-exfiltration guard.
//	I2 — An assertion minted for SP-A (Audience = A's EntityID) cannot validate at
//	     SP-B, because the Audience is bound to this record's EntityID.
//	I3 — IdP-initiated SSO is refused unless AllowIDPInitiated is true.
//
// Note: any field addition must also be reflected in the cassandradb provider,
// whose struct tags are authoritative for that driver.
type SAMLServiceProvider struct {
	Key string `json:"_key,omitempty" bson:"_key,omitempty" cql:"_key,omitempty" dynamo:"key,omitempty"` // ArangoDB document key

	ID string `gorm:"primaryKey;type:char(36)" json:"_id" bson:"_id" cql:"id" dynamo:"id,hash"`

	// OrgID scopes this registered SP to one Organization. Every IdP endpoint is
	// resolved per org_slug, so an SP registered for Org A is never reachable when
	// issuing for Org B. Immutable after creation.
	OrgID string `json:"org_id" bson:"org_id" cql:"org_id" dynamo:"org_id" gorm:"index" index:"org_id,hash"`

	// Name is a human-readable label (e.g. "Zendesk prod").
	Name string `json:"name" bson:"name" cql:"name" dynamo:"name"`

	// EntityID is the SP's SAML entity ID (the AuthnRequest Issuer value and the
	// assertion Audience). Unique within an org — the (org_id, entity_id) pair is
	// the lookup that resolves an incoming AuthnRequest to this record.
	EntityID string `json:"entity_id" bson:"entity_id" cql:"entity_id" dynamo:"entity_id"`

	// ACSURL is the SP's Assertion Consumer Service URL — the sole location a
	// signed assertion is POSTed to. NEVER taken from the request (I1).
	ACSURL string `json:"acs_url" bson:"acs_url" cql:"acs_url" dynamo:"acs_url"`

	// SPCertPEM is the SP's optional X.509 signing certificate (PEM). When present
	// it is used to encrypt-to the SP and (future) to validate signed
	// AuthnRequests. Optional — most SPs do not sign their AuthnRequests.
	SPCertPEM *string `json:"sp_cert_pem" bson:"sp_cert_pem" cql:"sp_cert_pem" dynamo:"sp_cert_pem"`

	// NameIDFormat is the SAML NameID format emitted in the Subject. Defaults to
	// urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress when empty.
	NameIDFormat string `json:"name_id_format" bson:"name_id_format" cql:"name_id_format" dynamo:"name_id_format"`

	// MappedAttributes is a JSON object mapping Authorizer profile fields to the
	// SAML attribute names this SP expects, e.g.
	// {"email":"email","given_name":"firstName","family_name":"lastName"}. This is
	// the inverse of the SP-side SAMLAttributeMapping: here the map's VALUES are
	// the emitted SAML attribute Names. Empty means "use default names".
	MappedAttributes *string `json:"mapped_attributes" bson:"mapped_attributes" cql:"mapped_attributes" dynamo:"mapped_attributes"`

	// AllowIDPInitiated permits unsolicited (IdP-initiated) SSO to this SP's ACS.
	// DEFAULT FALSE — SP-initiated only, which binds InResponseTo to a pending
	// AuthnRequest. Enable only when the SP supports unsolicited assertions.
	AllowIDPInitiated bool `json:"allow_idp_initiated" bson:"allow_idp_initiated" cql:"allow_idp_initiated" dynamo:"allow_idp_initiated"`

	// IsActive controls whether Authorizer will issue assertions to this SP.
	IsActive bool `json:"is_active" bson:"is_active" cql:"is_active" dynamo:"is_active" gorm:"default:true"`

	CreatedAt int64 `json:"created_at" bson:"created_at" cql:"created_at" dynamo:"created_at"`
	UpdatedAt int64 `json:"updated_at" bson:"updated_at" cql:"updated_at" dynamo:"updated_at"`
}

// AsAPISAMLServiceProvider converts the storage record into the GraphQL model.
func (s *SAMLServiceProvider) AsAPISAMLServiceProvider() *model.SAMLServiceProvider {
	id := s.ID
	if strings.Contains(id, Collections.SAMLServiceProvider+"/") {
		id = strings.TrimPrefix(id, Collections.SAMLServiceProvider+"/")
	}
	return &model.SAMLServiceProvider{
		ID:                id,
		OrgID:             s.OrgID,
		Name:              s.Name,
		EntityID:          s.EntityID,
		AcsURL:            s.ACSURL,
		SpCertPem:         s.SPCertPEM,
		NameIDFormat:      refs.NewStringRef(s.NameIDFormat),
		MappedAttributes:  s.MappedAttributes,
		AllowIdpInitiated: s.AllowIDPInitiated,
		IsActive:          s.IsActive,
		CreatedAt:         refs.NewInt64Ref(s.CreatedAt),
		UpdatedAt:         refs.NewInt64Ref(s.UpdatedAt),
	}
}
