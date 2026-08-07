package http_handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// jitStore is an in-memory storage.Provider stub covering only the methods
// jitProvisionSSOUser touches; every other method panics via the embedded nil.
type jitStore struct {
	storage.Provider
	federated      map[string]*schemas.FederatedIdentity // key: org|issuer|sub
	usersByID      map[string]*schemas.User
	usersByEmail   map[string]*schemas.User
	addUserCalls   int
	addFedCalls    int
	deleteUserCall int
	failAddFed     bool
}

func newJITStore() *jitStore {
	return &jitStore{
		federated:    map[string]*schemas.FederatedIdentity{},
		usersByID:    map[string]*schemas.User{},
		usersByEmail: map[string]*schemas.User{},
	}
}

func fedKey(org, iss, sub string) string { return org + "|" + iss + "|" + sub }

func (s *jitStore) GetFederatedIdentity(_ context.Context, org, iss, sub string) (*schemas.FederatedIdentity, error) {
	if fi, ok := s.federated[fedKey(org, iss, sub)]; ok {
		return fi, nil
	}
	return nil, errors.New("not found")
}

func (s *jitStore) GetUserByID(_ context.Context, id string) (*schemas.User, error) {
	if u, ok := s.usersByID[id]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func (s *jitStore) GetUserByEmail(_ context.Context, email string) (*schemas.User, error) {
	if u, ok := s.usersByEmail[email]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func (s *jitStore) AddUser(_ context.Context, user *schemas.User) (*schemas.User, error) {
	s.addUserCalls++
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	s.usersByID[user.ID] = user
	if user.Email != nil {
		s.usersByEmail[*user.Email] = user
	}
	return user, nil
}

func (s *jitStore) AddFederatedIdentity(_ context.Context, fi *schemas.FederatedIdentity) (*schemas.FederatedIdentity, error) {
	s.addFedCalls++
	if s.failAddFed {
		return nil, errors.New("simulated federated-identity insert failure")
	}
	if fi.ID == "" {
		fi.ID = uuid.New().String()
	}
	s.federated[fedKey(fi.OrgID, fi.Issuer, fi.Subject)] = fi
	return fi, nil
}

func (s *jitStore) DeleteUser(_ context.Context, user *schemas.User) error {
	s.deleteUserCall++
	delete(s.usersByID, user.ID)
	if user.Email != nil {
		delete(s.usersByEmail, *user.Email)
	}
	return nil
}

func (s *jitStore) AddOrgMembership(_ context.Context, m *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return m, nil
}

func newJITProvider(store *jitStore, signupEnabled bool) *httpProvider {
	logger := zerolog.Nop()
	return &httpProvider{
		Config:       &config.Config{EnableSignup: signupEnabled, DefaultRoles: []string{"user"}},
		Dependencies: Dependencies{Log: &logger, StorageProvider: store},
	}
}

func jitFlow() *ssoFlowState {
	return &ssoFlowState{OrgID: "org-1", ExpectedIssuer: ssoTestIssuer}
}

func jitClaims(sub, email string) jwt.MapClaims {
	c := jwt.MapClaims{"sub": sub, "email_verified": true}
	if email != "" {
		c["email"] = email
	}
	return c
}

// New federated principal → a fresh user + federated-identity row are created.
func TestSSOJIT_NewUserProvisioned(t *testing.T) {
	store := newJITStore()
	h := newJITProvider(store, true)

	user, isSignUp, err := h.jitProvisionSSOUser(context.Background(), jitFlow(), jitClaims("upstream-1", "alice@corp.example.com"))
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.True(t, isSignUp)
	assert.Equal(t, "alice@corp.example.com", refs.StringValue(user.Email))
	assert.Equal(t, 1, store.addUserCalls)
	assert.Equal(t, 1, store.addFedCalls)
}

// A returning federated principal is resolved via (org,issuer,sub) — NOT re-created.
func TestSSOJIT_ReturningUserResolvedByFederatedIdentity(t *testing.T) {
	store := newJITStore()
	h := newJITProvider(store, true)
	flow := jitFlow()

	first, _, err := h.jitProvisionSSOUser(context.Background(), flow, jitClaims("upstream-2", "bob@corp.example.com"))
	require.NoError(t, err)

	again, isSignUp, err := h.jitProvisionSSOUser(context.Background(), flow, jitClaims("upstream-2", "bob@corp.example.com"))
	require.NoError(t, err)
	assert.False(t, isSignUp)
	assert.Equal(t, first.ID, again.ID)
	assert.Equal(t, 1, store.addUserCalls, "returning principal must not create a second user")
}

// ACCOUNT-TAKEOVER DEFENSE: a federated login whose email collides with an
// existing (non-federated) account must NOT be silently linked — it is rejected
// fail-closed, and no user/federated row is created.
func TestSSOJIT_EmailCollisionNotLinked(t *testing.T) {
	store := newJITStore()
	// Pre-existing global account with the same email but a DIFFERENT identity.
	existing := &schemas.User{ID: "existing-user", Email: refs.NewStringRef("victim@corp.example.com")}
	store.usersByID[existing.ID] = existing
	store.usersByEmail["victim@corp.example.com"] = existing

	h := newJITProvider(store, true)
	user, isSignUp, err := h.jitProvisionSSOUser(context.Background(), jitFlow(), jitClaims("attacker-upstream", "victim@corp.example.com"))
	require.Error(t, err)
	assert.Nil(t, user)
	assert.False(t, isSignUp)
	assert.Contains(t, err.Error(), "already exists")
	assert.Equal(t, 0, store.addUserCalls, "must not create a shadow user")
	assert.Equal(t, 0, store.addFedCalls, "must not link a federated identity to the victim account")
}

// LOW-1: if AddFederatedIdentity fails after AddUser, the just-created user must
// be deleted (compensating action) so no orphan pollutes email lookups and the
// principal isn't locked out on retry.
func TestSSOJIT_OrphanUserCleanedUpOnFedIdentityFailure(t *testing.T) {
	store := newJITStore()
	store.failAddFed = true
	h := newJITProvider(store, true)

	user, _, err := h.jitProvisionSSOUser(context.Background(), jitFlow(), jitClaims("upstream-x", "dave@corp.example.com"))
	require.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, 1, store.addUserCalls)
	assert.Equal(t, 1, store.deleteUserCall, "the orphaned user must be deleted")
	assert.Empty(t, store.usersByEmail, "no orphan user may remain")
	assert.Empty(t, store.usersByID, "no orphan user may remain")
}

// Signup disabled → a first-time federated principal is rejected.
func TestSSOJIT_SignupDisabledRejected(t *testing.T) {
	store := newJITStore()
	h := newJITProvider(store, false)
	user, _, err := h.jitProvisionSSOUser(context.Background(), jitFlow(), jitClaims("upstream-3", "carol@corp.example.com"))
	require.Error(t, err)
	assert.Nil(t, user)
}

// REGRESSION: a revoked user reaching the SSO callback via the RETURNING-user
// path (resolved through an existing FederatedIdentity row, not first-time JIT
// provisioning) must be rejected, not silently handed a session. Confirms
// jitProvisionFederatedUser's RevokedTimestamp check fires on this path too.
func TestSSOJIT_RevokedReturningUserRejected(t *testing.T) {
	store := newJITStore()
	revokedAt := time.Now().Unix()
	user := &schemas.User{ID: "revoked-user", Email: refs.NewStringRef("evicted@corp.example.com"), RevokedTimestamp: &revokedAt}
	store.usersByID[user.ID] = user
	store.usersByEmail["evicted@corp.example.com"] = user
	store.federated[fedKey("org-1", ssoTestIssuer, "upstream-revoked")] = &schemas.FederatedIdentity{
		OrgID: "org-1", Issuer: ssoTestIssuer, Subject: "upstream-revoked", UserID: user.ID,
	}

	h := newJITProvider(store, true)
	got, isSignUp, err := h.jitProvisionSSOUser(context.Background(), jitFlow(), jitClaims("upstream-revoked", "evicted@corp.example.com"))
	require.Error(t, err)
	assert.Nil(t, got)
	assert.False(t, isSignUp)
	assert.Contains(t, err.Error(), "revoked")
}

// --- Federated principal-class matrix ---------------------------------------
//
// jitProvisionFederatedUser is the shared core for BOTH the per-org OIDC broker
// (SSO) and the SAML SP, so its invariants have to hold for both. They are NOT
// the same invariants the social (OAuth RP) path has, and conflating the two
// classes is exactly what produced the nOAuth takeover: social resolved an
// account by email, while this path — correctly — never does.
//
// Per class:
//
//	SSO / SAML  identity is (org, issuer, subject). Email NEVER selects an
//	            account, verified or not. A collision is refused outright.
//	Social      identity is the email, so the provider must attest it
//	            (oauth_noauth_test.go / the per-provider e2e specs).
//	Database    identity is the email, verified by clicking a mailed link.
//
// These pin the SSO/SAML column against drift toward the social one.

// TestFederatedMatrix_EmailNeverSelectsAnAccount is the column's defining
// property, asserted for every combination of attested and unattested email.
// Neither is allowed to link, because linking by email is not a thing this path
// does at all — the attestation is irrelevant here, unlike on the social path.
func TestFederatedMatrix_EmailNeverSelectsAnAccount(t *testing.T) {
	for _, tc := range []struct {
		name          string
		emailVerified bool
	}{
		{"attested upstream email", true},
		{"unattested upstream email", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newJITStore()
			// An account already holds the address, by some other credential.
			existing := &schemas.User{ID: "existing-1", Email: refs.NewStringRef("clash@corp.example.com")}
			store.usersByID[existing.ID] = existing
			store.usersByEmail["clash@corp.example.com"] = existing

			h := newJITProvider(store, true)
			claims := jwt.MapClaims{"sub": "upstream-clash", "email": "clash@corp.example.com", "email_verified": tc.emailVerified}

			_, _, err := h.jitProvisionSSOUser(context.Background(), jitFlow(), claims)
			require.Error(t, err,
				"a federated principal must never be merged into an account it did not prove it owns — "+
					"this path is keyed on (org, issuer, subject), so an email match is not evidence of anything")
			assert.Equal(t, 0, store.addUserCalls, "and nothing may be provisioned on the refusal path")
		})
	}
}

// TestFederatedMatrix_SubjectIsTheIdentity pins the flip side: the SAME email
// arriving under a DIFFERENT upstream subject is a different principal, and the
// same subject is the same principal regardless of what the email does.
func TestFederatedMatrix_SubjectIsTheIdentity(t *testing.T) {
	store := newJITStore()
	h := newJITProvider(store, true)

	first, isSignUp, err := h.jitProvisionSSOUser(context.Background(), jitFlow(), jitClaims("subject-a", "a@corp.example.com"))
	require.NoError(t, err)
	require.True(t, isSignUp)

	// Same subject, email changed upstream (a rename, or a mutable directory
	// attribute): still the same local account, resolved without touching email.
	again, isSignUp, err := h.jitProvisionSSOUser(context.Background(), jitFlow(), jitClaims("subject-a", "renamed@corp.example.com"))
	require.NoError(t, err)
	assert.False(t, isSignUp, "a returning subject must not be re-provisioned")
	assert.Equal(t, first.ID, again.ID, "the subject is the identity, not the email")
	assert.Equal(t, 1, store.addUserCalls)
}

// TestFederatedMatrix_OrgScopesTheIdentity pins tenant isolation: the same
// upstream subject under a DIFFERENT org is a different principal. Without this
// one tenant's IdP could mint principals inside another tenant.
func TestFederatedMatrix_OrgScopesTheIdentity(t *testing.T) {
	store := newJITStore()
	h := newJITProvider(store, true)

	_, _, err := h.jitProvisionSSOUser(context.Background(), &ssoFlowState{OrgID: "org-1", ExpectedIssuer: ssoTestIssuer},
		jitClaims("shared-subject", "user@corp.example.com"))
	require.NoError(t, err)

	// Same subject and issuer, different org. The email collides, so the
	// fail-closed guard fires rather than silently handing org-2 org-1's user.
	_, _, err = h.jitProvisionSSOUser(context.Background(), &ssoFlowState{OrgID: "org-2", ExpectedIssuer: ssoTestIssuer},
		jitClaims("shared-subject", "user@corp.example.com"))
	require.Error(t, err, "an identity is scoped to its org; org-2 must not resolve org-1's principal")
}
