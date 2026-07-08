package constants

// TrustedIssuer.Kind discriminator values (design §4.3 / §5 K1). Immutable after
// creation. The unified authorizer_trusted_issuers table serves two distinct
// trust relationships that MUST NOT be confused (design §5.2 CR1):
//
//   - client_assertion_trust: an external JWT issuer whose tokens authenticate an
//     OAuth *client* (RFC 7523 client_assertion). Subject-pinned, org-global.
//   - sso_oidc: a per-organization upstream OIDC IdP that Authorizer brokers as a
//     Relying Party. Org-scoped (OrgID set), NO subject pin — it federates end
//     users, it can never authenticate a client.
//   - sso_saml: reserved for the SAML Service Provider (separate PR).
const (
	// TrustKindClientAssertion is the default kind. Every pre-existing row (which
	// may have no kind column value) is treated as this kind on read
	// (TrustedIssuer.EffectiveKind), so an upgrade never breaks existing
	// client_assertion trust rows.
	TrustKindClientAssertion = "client_assertion_trust"

	// TrustKindSSOOIDC is a per-org upstream OIDC IdP brokered by Authorizer.
	TrustKindSSOOIDC = "sso_oidc"

	// TrustKindSSOSAML is reserved for the per-org SAML SP connection.
	// ponytail: reserved only — SAML SP (signature-wrapping/XSW validation) ships
	// in its own PR so it gets a focused security review.
	TrustKindSSOSAML = "sso_saml"
)
