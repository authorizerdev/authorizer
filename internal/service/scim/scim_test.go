package scim

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
	"github.com/authorizerdev/authorizer/internal/graph/model"
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

func (f *fakeStore) GetUserByPhoneNumber(_ context.Context, phone string) (*schemas.User, error) {
	for _, u := range f.users {
		if u.PhoneNumber != nil && *u.PhoneNumber == phone {
			return u, nil
		}
	}
	return nil, errNotFound
}

// ListOrgMembershipsByOrg returns the org's memberships in a stable order,
// honouring the pagination offset/limit (enough for the ListUsers scan tests).
func (f *fakeStore) ListOrgMembershipsByOrg(_ context.Context, orgID string, p *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	var keys []string
	for k, ok := range f.memberships {
		if ok && strings.HasPrefix(k, orgID+"|") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	total := int64(len(keys))
	from := int(p.Offset)
	if from > len(keys) {
		from = len(keys)
	}
	to := from + int(p.Limit)
	if to > len(keys) {
		to = len(keys)
	}
	var out []*schemas.OrgMembership
	for _, k := range keys[from:to] {
		out = append(out, &schemas.OrgMembership{OrgID: orgID, UserID: strings.SplitN(k, "|", 2)[1]})
	}
	return out, &model.Pagination{Limit: p.Limit, Page: p.Page, Offset: p.Offset, Total: total}, nil
}

// spyEvents records the provisioning-lifecycle events the service fires. Because
// events go through asyncutil.Go, tests must asyncutil.Wait before asserting.
type spyEvents struct {
	mu          sync.Mutex
	userEvents  []string
	groupEvents []string
}

func (s *spyEvents) RegisterEvent(_ context.Context, eventName string, _ string, _ *schemas.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userEvents = append(s.userEvents, eventName)
	return nil
}

func (s *spyEvents) RegisterScimGroupEvent(_ context.Context, eventName string, _ *schemas.ScimGroup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groupEvents = append(s.groupEvents, eventName)
	return nil
}

func (s *spyEvents) users() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.userEvents...)
}

func (s *spyEvents) groups() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.groupEvents...)
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

// seedUser inserts a user into the fake store and binds them to org.
func seedUser(store *fakeStore, org, id, email, given, family, ext string, active bool) {
	u := &schemas.User{ID: id, IsActive: active}
	if email != "" {
		u.Email = &email
	}
	if given != "" {
		u.GivenName = &given
	}
	if family != "" {
		u.FamilyName = &family
	}
	if ext != "" {
		ns := org + ":" + ext
		u.ExternalID = &ns
	}
	store.users[id] = u
	store.memberships[org+"|"+id] = true
}

func ids(users []*schemas.User) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		out = append(out, u.ID)
	}
	return out
}

// TestListUsers_Filters exercises every supported operator against every
// supported attribute, plus a negative (matches-nothing) case and cross-org
// isolation.
func TestListUsers_Filters(t *testing.T) {
	p, store, _ := newSvc(t)
	ctx := context.Background()
	const org = "org-a"
	seedUser(store, org, "u1", "bob@acme.com", "Bob", "Doe", "okta-1", true)
	seedUser(store, org, "u2", "alice@acme.com", "Alice", "Smith", "", false)
	// A member of a different org with a colliding givenName — must never surface.
	seedUser(store, "org-b", "b1", "bob@other.com", "Bob", "Doe", "okta-b", true)

	cases := []struct {
		name string
		f    UserFilter
		want []string
	}{
		{"eq userName (indexed)", UserFilter{"userName", "eq", "bob@acme.com"}, []string{"u1"}},
		{"eq emails.value (indexed)", UserFilter{"emails.value", "eq", "alice@acme.com"}, []string{"u2"}},
		{"eq externalId (indexed)", UserFilter{"externalId", "eq", "okta-1"}, []string{"u1"}},
		{"eq userName no match", UserFilter{"userName", "eq", "nobody@acme.com"}, nil},
		{"ne givenName", UserFilter{"name.givenName", "ne", "bob"}, []string{"u2"}},
		{"co familyName", UserFilter{"name.familyName", "co", "oe"}, []string{"u1"}},
		{"sw givenName", UserFilter{"name.givenName", "sw", "al"}, []string{"u2"}},
		{"pr externalId", UserFilter{"externalId", "pr", ""}, []string{"u1"}},
		{"active eq true", UserFilter{"active", "eq", "true"}, []string{"u1"}},
		{"active eq false", UserFilter{"active", "eq", "false"}, []string{"u2"}},
		{"active ne true", UserFilter{"active", "ne", "true"}, []string{"u2"}},
		{"co familyName matches nothing", UserFilter{"name.familyName", "co", "zzz"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := p.ListUsers(ctx, org, tc.f)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.want, ids(got))
		})
	}
}

// drainEvents waits for asyncutil.Go-dispatched webhook goroutines to finish so
// the spy's recorded events are visible.
func drainEvents() { asyncutil.Wait(zerolog.Nop()) }

func newSvcWithEvents(t *testing.T) (*provider, *fakeStore, *spyEvents) {
	t.Helper()
	p, store, _ := newSvc(t)
	spy := &spyEvents{}
	p.EventsProvider = spy
	return p, store, spy
}

// TestScimUserWebhookEvents proves each user lifecycle event fires exactly once
// on its triggering operation (and not on idempotent no-ops).
func TestScimUserWebhookEvents(t *testing.T) {
	p, _, spy := newSvcWithEvents(t)
	ctx := context.Background()
	const org = "org-a"

	// Create → user.provisioned.
	u, _, err := p.CreateUser(ctx, org, User{ExternalID: "e1", UserName: "bob@acme.com", Active: true})
	require.NoError(t, err)
	drainEvents()
	assert.Equal(t, []string{"user.provisioned"}, spy.users())

	// Attribute change via PATCH → user.scim_updated.
	gn := "Robert"
	_, err = p.PatchUser(ctx, org, u.ID, UserPatch{GivenName: &gn})
	require.NoError(t, err)
	drainEvents()
	assert.Equal(t, []string{"user.provisioned", "user.scim_updated"}, spy.users())

	// Deactivate via SetActive(false) → user.deprovisioned.
	_, err = p.SetActive(ctx, org, u.ID, false)
	require.NoError(t, err)
	drainEvents()
	assert.Equal(t, []string{"user.provisioned", "user.scim_updated", "user.deprovisioned"}, spy.users())

	// Idempotent SetActive(false) again → no new event.
	_, err = p.SetActive(ctx, org, u.ID, false)
	require.NoError(t, err)
	drainEvents()
	assert.Equal(t, []string{"user.provisioned", "user.scim_updated", "user.deprovisioned"}, spy.users(),
		"a no-op deactivate must not re-fire")
}

// TestScimGroupWebhookEvents proves the three group events fire on create,
// update (membership), and delete.
func TestScimGroupWebhookEvents(t *testing.T) {
	p, store := newGroupSvc(t)
	spy := &spyEvents{}
	p.EventsProvider = spy
	ctx := context.Background()
	const org = "org-a"
	store.memberships[org+"|u1"] = true

	g, _, err := p.CreateGroup(ctx, org, Group{DisplayName: "Engineers", ExternalID: "ext-1"})
	require.NoError(t, err)
	drainEvents()
	assert.Equal(t, []string{"group.created"}, spy.groups())

	_, err = p.PatchGroup(ctx, org, g.ID, nil, nil, []MemberOp{{Op: "add", Members: []string{"u1"}}})
	require.NoError(t, err)
	drainEvents()
	assert.Equal(t, []string{"group.created", "group.updated"}, spy.groups())

	require.NoError(t, p.DeleteGroup(ctx, org, g.ID))
	drainEvents()
	assert.Equal(t, []string{"group.created", "group.updated", "group.deleted"}, spy.groups())
}

// TestPatchUser_Attributes covers each PATCH-able attribute, the duplicate-email
// rejection, and the no-op (nothing changed) case.
func TestPatchUser_Attributes(t *testing.T) {
	p, store, _ := newSvc(t)
	ctx := context.Background()
	const org = "org-a"
	u, _, err := p.CreateUser(ctx, org, User{ExternalID: "e1", UserName: "bob@acme.com", GivenName: "Bob", Active: true})
	require.NoError(t, err)

	t.Run("name/phone/externalId", func(t *testing.T) {
		gn, fn, phone, ext := "Robert", "Doe", "+15551234567", "okta-new"
		got, err := p.PatchUser(ctx, org, u.ID, UserPatch{GivenName: &gn, FamilyName: &fn, PhoneNumber: &phone, ExternalID: &ext})
		require.NoError(t, err)
		assert.Equal(t, "Robert", *got.GivenName)
		assert.Equal(t, "Doe", *got.FamilyName)
		assert.Equal(t, "+15551234567", *got.PhoneNumber)
		assert.Equal(t, "org-a:okta-new", *got.ExternalID, "externalId must be re-namespaced")
	})

	t.Run("email change", func(t *testing.T) {
		email := "robert@acme.com"
		got, err := p.PatchUser(ctx, org, u.ID, UserPatch{Email: &email})
		require.NoError(t, err)
		assert.Equal(t, "robert@acme.com", *got.Email)
	})

	t.Run("duplicate email rejected", func(t *testing.T) {
		// Another user owns this email.
		seedUser(store, org, "u2", "taken@acme.com", "", "", "", true)
		email := "taken@acme.com"
		_, err := p.PatchUser(ctx, org, u.ID, UserPatch{Email: &email})
		assert.ErrorIs(t, err, ErrConflict, "a PATCH must not set an email another user holds")
	})

	t.Run("no-op patch changes nothing", func(t *testing.T) {
		gn := "Robert" // already set to Robert above
		before := *store.users[u.ID]
		got, err := p.PatchUser(ctx, org, u.ID, UserPatch{GivenName: &gn})
		require.NoError(t, err)
		assert.Equal(t, before.UpdatedAt, got.UpdatedAt, "an idempotent patch must not bump updated_at")
	})

	t.Run("active:false deactivates and revokes", func(t *testing.T) {
		active := false
		got, err := p.PatchUser(ctx, org, u.ID, UserPatch{Active: &active})
		require.NoError(t, err)
		assert.False(t, got.IsActive)
		require.NotNil(t, got.RevokedTimestamp)
	})
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
