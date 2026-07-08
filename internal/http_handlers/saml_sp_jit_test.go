package http_handlers

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/memory_store/in_memory"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Profile attributes are extracted from the assertion using the connection's
// custom attribute mapping; the NameID stays the federated subject.
func TestSAML_ProfileExtractionWithCustomMapping(t *testing.T) {
	idp := newSAMLIdP(t)
	p := defaultAssertionParams()
	p.attributes = map[string]string{"mail": "custom@corp.example.com", "gn": "Ada"}
	assertion := parseValidAssertion(t, idp, p)

	prof := extractSAMLProfile(assertion, refs.NewStringRef(`{"email":"mail","given_name":"gn"}`))
	assert.Equal(t, "custom@corp.example.com", prof.Email)
	assert.Equal(t, "Ada", prof.GivenName)
}

// With the default mapping, the standard "email" attribute is picked up.
func TestSAML_ProfileExtractionDefaultMapping(t *testing.T) {
	idp := newSAMLIdP(t)
	assertion := parseValidAssertion(t, idp, defaultAssertionParams())
	prof := extractSAMLProfile(assertion, nil)
	assert.Equal(t, samlTestEmail, prof.Email)
}

// SAML email-collision: a first-time NameID whose mapped email collides with an
// existing account is rejected fail-closed and never linked (account takeover).
func TestSAML_EmailCollisionNotLinked(t *testing.T) {
	store := newJITStore()
	existing := &schemas.User{ID: "existing-1", Email: refs.NewStringRef(samlTestEmail)}
	store.usersByID[existing.ID] = existing
	store.usersByEmail[samlTestEmail] = existing
	h := newJITProvider(store, true)

	profile := federatedProfile{Email: samlTestEmail, EmailVerified: true}
	user, isSignUp, err := h.jitProvisionFederatedUser(context.Background(), samlOrgAID, samlIdPEntityID, samlTestNameID, profile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Nil(t, user)
	assert.False(t, isSignUp)
	assert.Equal(t, 0, store.addFedCalls, "no federated identity may be created on collision")
}

// SAML replay: a second presentation of the same AssertionID within an org is
// rejected by the single-use cache; a different org is independent (namespaced).
func TestSAML_AssertionReplayRejected(t *testing.T) {
	logger := zerolog.Nop()
	mem, err := in_memory.NewInMemoryProvider(&config.Config{}, &in_memory.Dependencies{Log: &logger})
	require.NoError(t, err)
	h := &httpProvider{
		Config:       &config.Config{},
		Dependencies: Dependencies{Log: &logger, MemoryStoreProvider: mem},
	}
	idp := newSAMLIdP(t)
	assertion := parseValidAssertion(t, idp, defaultAssertionParams())

	require.NoError(t, h.consumeSAMLAssertionID(samlOrgAID, assertion))
	require.Error(t, h.consumeSAMLAssertionID(samlOrgAID, assertion))
	require.NoError(t, h.consumeSAMLAssertionID("other-org", assertion))
}
