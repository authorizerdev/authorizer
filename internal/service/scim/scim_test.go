package scim

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// fakeStore is a stateful in-memory storage.Provider covering only the methods
// the SCIM service touches. Every other method panics via the embedded nil.
type fakeStore struct {
	storage.Provider
	endpoints   map[string]*schemas.ScimEndpoint // by id
	users       map[string]*schemas.User         // by id
	memberships map[string]bool                  // "orgID|userID"
	groups      map[string]*schemas.ScimGroup    // by id
	deletedTok  []string                         // user ids passed to DeleteAllSessionTokensByUserID
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		endpoints:   map[string]*schemas.ScimEndpoint{},
		users:       map[string]*schemas.User{},
		memberships: map[string]bool{},
		groups:      map[string]*schemas.ScimGroup{},
	}
}

var errNotFound = errors.New("not found")

func (f *fakeStore) GetScimEndpointByID(_ context.Context, id string) (*schemas.ScimEndpoint, error) {
	if e, ok := f.endpoints[id]; ok {
		return e, nil
	}
	return nil, errNotFound
}

func (f *fakeStore) GetUserByExternalID(_ context.Context, orgID, externalID string) (*schemas.User, error) {
	key := orgID + ":" + externalID
	for _, u := range f.users {
		if u.ExternalID != nil && *u.ExternalID == key {
			return u, nil
		}
	}
	return nil, errNotFound
}

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (*schemas.User, error) {
	for _, u := range f.users {
		if u.Email != nil && *u.Email == email {
			return u, nil
		}
	}
	return nil, errNotFound
}

func (f *fakeStore) GetUserByID(_ context.Context, id string) (*schemas.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, errNotFound
}

func (f *fakeStore) AddUser(_ context.Context, u *schemas.User) (*schemas.User, error) {
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeStore) UpdateUser(_ context.Context, u *schemas.User) (*schemas.User, error) {
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeStore) GetOrgMembership(_ context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	if f.memberships[orgID+"|"+userID] {
		return &schemas.OrgMembership{OrgID: orgID, UserID: userID}, nil
	}
	return nil, errNotFound
}

func (f *fakeStore) AddOrgMembership(_ context.Context, m *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	f.memberships[m.OrgID+"|"+m.UserID] = true
	return m, nil
}

func (f *fakeStore) DeleteAllSessionTokensByUserID(_ context.Context, userID string) error {
	f.deletedTok = append(f.deletedTok, userID)
	return nil
}

// fakeMem records the user ids whose sessions were dropped.
type fakeMem struct {
	memory_store.Provider
	deleted []string
}

func (m *fakeMem) DeleteAllUserSessions(userID string) error {
	m.deleted = append(m.deleted, userID)
	return nil
}

func newSvc(t *testing.T) (*provider, *fakeStore, *fakeMem) {
	t.Helper()
	log := zerolog.Nop()
	store := newFakeStore()
	mem := &fakeMem{}
	p := &provider{Dependencies: Dependencies{Log: &log, StorageProvider: store, MemoryStoreProvider: mem}}
	return p, store, mem
}

// seedEndpoint inserts an endpoint whose bearer secret is `secret`.
func seedEndpoint(t *testing.T, store *fakeStore, id, orgID, secret string, enabled bool) {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(secret), tokenCost)
	require.NoError(t, err)
	store.endpoints[id] = &schemas.ScimEndpoint{ID: id, OrgID: orgID, TokenHash: string(h), Enabled: enabled}
}

func TestAuthenticate(t *testing.T) {
	p, store, _ := newSvc(t)
	seedEndpoint(t, store, "ep-a", "org-a", "s3cr3t", true)
	seedEndpoint(t, store, "ep-disabled", "org-x", "s3cr3t", false)
	ctx := context.Background()

	t.Run("valid token resolves org from the endpoint only", func(t *testing.T) {
		org, err := p.Authenticate(ctx, "ep-a.s3cr3t")
		require.NoError(t, err)
		assert.Equal(t, "org-a", org)
	})
	t.Run("wrong secret rejected", func(t *testing.T) {
		_, err := p.Authenticate(ctx, "ep-a.wrong")
		assert.ErrorIs(t, err, ErrUnauthorized)
	})
	t.Run("unknown endpoint id rejected", func(t *testing.T) {
		_, err := p.Authenticate(ctx, "ep-nope.s3cr3t")
		assert.ErrorIs(t, err, ErrUnauthorized)
	})
	t.Run("malformed token rejected", func(t *testing.T) {
		for _, tok := range []string{"", "no-dot", "ep-a.", ".secret"} {
			_, err := p.Authenticate(ctx, tok)
			assert.ErrorIs(t, err, ErrUnauthorized, tok)
		}
	})
	t.Run("disabled endpoint rejected", func(t *testing.T) {
		_, err := p.Authenticate(ctx, "ep-disabled.s3cr3t")
		assert.ErrorIs(t, err, ErrUnauthorized)
	})
}

func TestGenerateToken(t *testing.T) {
	tok, hash, err := GenerateToken("ep-1")
	require.NoError(t, err)
	// Format: "<id>.<hexSecret>", 64 hex chars (256 bits).
	assert.Regexp(t, `^ep-1\.[0-9a-f]{64}$`, tok)
	secret := tok[len("ep-1."):]
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)))
}

func TestCreateUser_ProvisionsWithNamespacedExternalID(t *testing.T) {
	p, store, _ := newSvc(t)
	ctx := context.Background()
	u, existed, err := p.CreateUser(ctx, "org-a", User{ExternalID: "okta-123", UserName: "bob@acme.com", GivenName: "Bob", Active: true})
	require.NoError(t, err)
	assert.False(t, existed)
	require.NotNil(t, u.ExternalID)
	assert.Equal(t, "org-a:okta-123", *u.ExternalID, "external_id must be org-namespaced")
	assert.True(t, u.IsActive)
	// Membership binds the user to the org (required for later by-id ops).
	_, err = store.GetOrgMembership(ctx, "org-a", u.ID)
	assert.NoError(t, err)
}

func TestCreateUser_ProvisionInactiveIsRevoked(t *testing.T) {
	p, _, _ := newSvc(t)
	ctx := context.Background()
	u, _, err := p.CreateUser(ctx, "org-a", User{ExternalID: "okta-9", UserName: "eve@acme.com", Active: false})
	require.NoError(t, err)
	assert.False(t, u.IsActive, "a user provisioned active:false must be stored inactive")
	require.NotNil(t, u.RevokedTimestamp, "an inactive-provisioned user must be revocation-stamped so no token can be minted")
}

func TestCreateUser_DedupByExternalID(t *testing.T) {
	p, _, _ := newSvc(t)
	ctx := context.Background()
	first, _, err := p.CreateUser(ctx, "org-a", User{ExternalID: "okta-123", UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)
	second, existed, err := p.CreateUser(ctx, "org-a", User{ExternalID: "okta-123", UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)
	assert.True(t, existed, "repeat create with same externalId must dedup")
	assert.Equal(t, first.ID, second.ID, "no duplicate user")
}

func TestCreateUser_DedupByUserNameWithinOrg(t *testing.T) {
	p, _, _ := newSvc(t)
	ctx := context.Background()
	first, _, err := p.CreateUser(ctx, "org-a", User{UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)
	second, existed, err := p.CreateUser(ctx, "org-a", User{UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)
	assert.True(t, existed)
	assert.Equal(t, first.ID, second.ID)
}

func TestCreateUser_CrossOrgEmailConflict(t *testing.T) {
	p, _, _ := newSvc(t)
	ctx := context.Background()
	_, _, err := p.CreateUser(ctx, "org-a", User{UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)
	// A different org tries to provision the same email → global email
	// uniqueness means we cannot, and we must not silently adopt org-a's user.
	_, _, err = p.CreateUser(ctx, "org-b", User{UserName: "bob@acme.com", Active: true})
	assert.ErrorIs(t, err, ErrConflict)
}

// TestH6_CrossOrgMutationRejected is the core cross-org isolation guarantee:
// org-a's SCIM connection can never read or mutate a user provisioned into
// org-b, even with that user's exact id.
func TestH6_CrossOrgMutationRejected(t *testing.T) {
	p, _, mem := newSvc(t)
	ctx := context.Background()
	victim, _, err := p.CreateUser(ctx, "org-b", User{ExternalID: "b-1", UserName: "victim@b.com", Active: true})
	require.NoError(t, err)

	t.Run("GET by id from another org 404s", func(t *testing.T) {
		_, err := p.GetUser(ctx, "org-a", victim.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("PATCH active:false from another org 404s and does NOT revoke", func(t *testing.T) {
		_, err := p.SetActive(ctx, "org-a", victim.ID, false)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Empty(t, mem.deleted, "a cross-org deprovision must not touch the victim's sessions")
		got, _ := p.GetUser(ctx, "org-b", victim.ID)
		assert.True(t, got.IsActive, "victim must remain active")
	})
	t.Run("PUT replace from another org 404s", func(t *testing.T) {
		_, err := p.ReplaceUser(ctx, "org-a", victim.ID, User{UserName: "victim@b.com", Active: false})
		assert.ErrorIs(t, err, ErrNotFound)
	})
	t.Run("owning org CAN mutate", func(t *testing.T) {
		u, err := p.SetActive(ctx, "org-b", victim.ID, false)
		require.NoError(t, err)
		assert.False(t, u.IsActive)
	})
}

// TestDeactivateRevokesSessions proves the enterprise-offboarding contract:
// active:false disables the user AND synchronously drops sessions + refresh/
// session tokens so held credentials stop working immediately.
func TestDeactivateRevokesSessions(t *testing.T) {
	p, store, mem := newSvc(t)
	ctx := context.Background()
	u, _, err := p.CreateUser(ctx, "org-a", User{ExternalID: "a-1", UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)

	got, err := p.SetActive(ctx, "org-a", u.ID, false)
	require.NoError(t, err)
	assert.False(t, got.IsActive)
	require.NotNil(t, got.RevokedTimestamp, "revoked_timestamp must be stamped so already-issued tokens fail")
	assert.Contains(t, mem.deleted, u.ID, "memory-store sessions must be dropped")
	assert.Contains(t, store.deletedTok, u.ID, "db session/refresh tokens must be dropped")
}

func TestDelete_MapsToDeactivate(t *testing.T) {
	p, _, mem := newSvc(t)
	ctx := context.Background()
	u, _, err := p.CreateUser(ctx, "org-a", User{UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)
	// DELETE is modelled as SetActive(false) at the handler; assert the same
	// revocation happens through the service path it calls.
	_, err = p.SetActive(ctx, "org-a", u.ID, false)
	require.NoError(t, err)
	assert.Contains(t, mem.deleted, u.ID)
}

func TestFindByUserName_OrgScoped(t *testing.T) {
	p, _, _ := newSvc(t)
	ctx := context.Background()
	u, _, err := p.CreateUser(ctx, "org-b", User{UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)
	// Same email, probed from another org → not surfaced (H6 for the dedup probe).
	got, err := p.FindByUserName(ctx, "org-a", "bob@acme.com")
	require.NoError(t, err)
	assert.Nil(t, got)
	// Owning org sees it.
	got, err = p.FindByUserName(ctx, "org-b", "bob@acme.com")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID, got.ID)
}
