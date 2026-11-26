package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// The doubles below emulate losing a create race: the uniqueness pre-check
// misses, the write then fails with the backend's own conflict error (masked as
// a storage error), and the post-failure probe finds the row a concurrent
// request created in between.

var backendConflict = &storage.Error{Err: errors.New("user with given email already exists")}

// racingStorage reports raceEmail as absent until AddUser has been attempted
// for it, then as present — exactly what a concurrent create looks like.
type racingStorage struct {
	*inviteStorage
	raceEmail   string
	existsFirst bool // the email exists before the request (pre-check path)
	raced       bool
}

func (s *racingStorage) GetUserByEmail(ctx context.Context, email string) (*schemas.User, error) {
	if email == s.raceEmail && (s.raced || s.existsFirst) {
		return &schemas.User{ID: "concurrent", Email: refs.NewStringRef(email)}, nil
	}
	return s.inviteStorage.GetUserByEmail(ctx, email)
}

func (s *racingStorage) GetUserByPhoneNumber(context.Context, string) (*schemas.User, error) {
	return nil, errors.New("user not found")
}

func (s *racingStorage) AddUser(ctx context.Context, u *schemas.User) (*schemas.User, error) {
	if refs.StringValue(u.Email) == s.raceEmail {
		s.raced = true
		return u, backendConflict
	}
	return s.inviteStorage.AddUser(ctx, u)
}

// TestInviteMembers_RaceSkipsEmail: an invitee created concurrently is skipped
// like the pre-check skips existing emails, instead of failing the request
// after earlier invitees were already created and emailed.
func TestInviteMembers_RaceSkipsEmail(t *testing.T) {
	st := &racingStorage{inviteStorage: &inviteStorage{}, raceEmail: "race@authorizer.dev"}
	p := newInviteProvider(inviteConfig(), st, inviteToken{})

	emails := []string{"a@authorizer.dev", "race@authorizer.dev", "b@authorizer.dev"}
	res, _, err := p.InviteMembers(context.Background(), inviteMeta(), &model.InviteMemberRequest{Emails: emails})
	require.NoError(t, err)
	require.Len(t, res.Users, 2)
	for _, u := range res.Users {
		assert.NotEqual(t, "race@authorizer.dev", refs.StringValue(u.Email))
	}
	assert.Equal(t, 2, st.addVerificationCalls, "no verification request for the skipped email")
}

// TestInviteMembers_AddUserOutageStaysInternal: if the probe finds nothing,
// the failure was not a conflict and the storage error is returned (masked).
func TestInviteMembers_AddUserOutageStaysInternal(t *testing.T) {
	st := &racingStorage{inviteStorage: &inviteStorage{}, raceEmail: "race@authorizer.dev"}
	outage := &outageStorage{racingStorage: st}
	p := newInviteProvider(inviteConfig(), outage, inviteToken{})

	_, _, err := p.InviteMembers(context.Background(), inviteMeta(), &model.InviteMemberRequest{Emails: []string{"race@authorizer.dev"}})
	require.Error(t, err)
	_, _, masked := MaskInternalError(err)
	assert.True(t, masked)
}

// outageStorage fails AddUser without the row ever appearing.
type outageStorage struct{ *racingStorage }

func (s *outageStorage) AddUser(_ context.Context, u *schemas.User) (*schemas.User, error) {
	return u, &storage.Error{Err: errors.New("connection reset by peer")}
}

// TestSignUp_EmailRaceKeepsAntiEnumerationAnswer: with email verification on,
// a lost race on the email must produce exactly the answer the email
// pre-check gives for a taken address, so the race window is not an
// account-existence oracle.
func TestSignUp_EmailRaceKeepsAntiEnumerationAnswer(t *testing.T) {
	cfg := inviteConfig()
	cfg.EnableSignup = true
	cfg.EnableEmailVerification = true
	cfg.IsEmailServiceEnabled = true
	req := func() *model.SignUpRequest {
		return &model.SignUpRequest{
			Email:           refs.NewStringRef("race@authorizer.dev"),
			Password:        "Password@123",
			ConfirmPassword: "Password@123",
		}
	}

	preSt := &racingStorage{inviteStorage: &inviteStorage{}, raceEmail: "race@authorizer.dev", existsFirst: true}
	preRes, _, preErr := newInviteProvider(cfg, preSt, inviteToken{}).SignUp(context.Background(), inviteMeta(), req())
	require.NoError(t, preErr)
	require.NotNil(t, preRes)

	raceSt := &racingStorage{inviteStorage: &inviteStorage{}, raceEmail: "race@authorizer.dev"}
	raceRes, _, raceErr := newInviteProvider(cfg, raceSt, inviteToken{}).SignUp(context.Background(), inviteMeta(), req())
	require.NoError(t, raceErr)
	require.True(t, raceSt.raced, "the race path must have been exercised")
	assert.Equal(t, preRes, raceRes)
}

// orgRaceStorage emulates losing a create-organization / add-member race.
type orgRaceStorage struct {
	*inviteStorage
	orgCreated    bool
	memberCreated bool
}

func (s *orgRaceStorage) GetOrganizationByName(_ context.Context, name string) (*schemas.Organization, error) {
	if s.orgCreated {
		return &schemas.Organization{ID: "concurrent", Name: name}, nil
	}
	return nil, errors.New("organization not found")
}

func (s *orgRaceStorage) AddOrganization(_ context.Context, org *schemas.Organization) (*schemas.Organization, error) {
	s.orgCreated = true
	return org, &storage.Error{Err: errors.New("organization with acme name already exists")}
}

func (s *orgRaceStorage) GetOrganizationByID(_ context.Context, id string) (*schemas.Organization, error) {
	return &schemas.Organization{ID: id, Name: "acme"}, nil
}

func (s *orgRaceStorage) GetUserByID(_ context.Context, id string) (*schemas.User, error) {
	return &schemas.User{ID: id}, nil
}

func (s *orgRaceStorage) GetOrgMembership(_ context.Context, orgID, userID string) (*schemas.OrgMembership, error) {
	if s.memberCreated {
		return &schemas.OrgMembership{OrgID: orgID, UserID: userID}, nil
	}
	return nil, errors.New("membership not found")
}

func (s *orgRaceStorage) AddOrgMembership(_ context.Context, m *schemas.OrgMembership) (*schemas.OrgMembership, error) {
	s.memberCreated = true
	return m, &storage.Error{Err: errors.New("membership for org_id o and user_id u already exists")}
}

func TestCreateOrganization_RaceIsAlreadyExists(t *testing.T) {
	p := newInviteProvider(inviteConfig(), &orgRaceStorage{inviteStorage: &inviteStorage{}}, inviteToken{})
	_, _, err := p.CreateOrganization(context.Background(), inviteMeta(), &model.CreateOrganizationRequest{Name: "acme"})
	requireKind(t, err, KindAlreadyExists)
	assert.Equal(t, "an organization with this name already exists", err.Error())
}

func TestAddOrgMember_RaceIsAlreadyExists(t *testing.T) {
	p := newInviteProvider(inviteConfig(), &orgRaceStorage{inviteStorage: &inviteStorage{}}, inviteToken{})
	_, _, err := p.AddOrgMember(context.Background(), inviteMeta(), &model.AddOrgMemberRequest{OrgID: "o", UserID: "u"})
	requireKind(t, err, KindAlreadyExists)
	assert.Equal(t, "user is already a member of this organization", err.Error())
}
