package clientauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestClientAssertion_SSORowRejected proves design §5.2 CR1: a per-org sso_oidc
// TrustedIssuer row registered at the SAME issuer URL must NEVER be usable to
// authenticate an OAuth client on the client_assertion path — even when the
// presented assertion is otherwise perfectly valid and signed by a key the row's
// JWKS serves. The kind/org guard in resolveViaClientAssertion rejects it.
func TestClientAssertion_SSORowRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)

	// Convert the trust row into a per-org sso_oidc connection at the same URL.
	store := r.StorageProvider.(*assertionStore)
	store.issuers[testIssuerURL].Kind = constants.TrustKindSSOOIDC
	store.issuers[testIssuerURL].OrgID = "org-123"

	client, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidClient, "an sso_oidc row must not authenticate a client")
	assert.Nil(t, client)
}

// TestClientAssertion_OrgScopedRowRejected proves the org_id half of the CR1
// guard: even a client_assertion_trust-kinded row is rejected if it is org-scoped
// (OrgID set) — instance-global rows only may authenticate clients.
func TestClientAssertion_OrgScopedRowRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)

	store := r.StorageProvider.(*assertionStore)
	store.issuers[testIssuerURL].Kind = constants.TrustKindClientAssertion
	store.issuers[testIssuerURL].OrgID = "org-456"

	client, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidClient, "an org-scoped trust row must not authenticate a client")
	assert.Nil(t, client)
}

// TestClientAssertion_EmptyKindStillAuthenticates guards the upgrade path: a row
// written before the kind column existed (empty Kind, empty OrgID) is treated as
// client_assertion_trust by EffectiveKind and MUST still authenticate.
func TestClientAssertion_EmptyKindStillAuthenticates(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)

	store := r.StorageProvider.(*assertionStore)
	store.issuers[testIssuerURL].Kind = "" // pre-migration row
	store.issuers[testIssuerURL].OrgID = ""

	client, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, testSAClientPK, client.ID)
}
