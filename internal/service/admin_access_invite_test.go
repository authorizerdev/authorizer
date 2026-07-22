package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// --- test doubles ---------------------------------------------------------
//
// Each embeds the full interface so only the handful of methods InviteMembers /
// the representative admin methods touch need overriding; any other call panics
// (nil embedded interface), which is the desired "this test should not reach
// here" behaviour.

type inviteStorage struct {
	storage.Provider
	getUserByEmailCalls  int
	addUserCalls         int
	addVerificationCalls int
	userExists           func(email string) bool
	getOrgByName         func(name string) *schemas.Organization
}

func (s *inviteStorage) GetUserByEmail(_ context.Context, email string) (*schemas.User, error) {
	s.getUserByEmailCalls++
	if s.userExists != nil && s.userExists(email) {
		return &schemas.User{ID: "existing-" + email, Email: refs.NewStringRef(email)}, nil
	}
	return nil, errors.New("user not found")
}

func (s *inviteStorage) AddUser(_ context.Context, u *schemas.User) (*schemas.User, error) {
	s.addUserCalls++
	// Emulate the storage layer assigning a persisted id so the test can prove
	// the response is sourced from AddUser's return value, not a re-fetch.
	u.ID = "id-" + refs.StringValue(u.Email)
	return u, nil
}

func (s *inviteStorage) AddVerificationRequest(_ context.Context, v *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	s.addVerificationCalls++
	return v, nil
}

func (s *inviteStorage) GetOrganizationByName(_ context.Context, name string) (*schemas.Organization, error) {
	if s.getOrgByName != nil {
		if org := s.getOrgByName(name); org != nil {
			return org, nil
		}
	}
	return nil, errors.New("organization not found")
}

type inviteToken struct {
	token.Provider
	createVerificationToken func(cfg *token.AuthTokenConfig) (string, error)
}

func (inviteToken) IsSuperAdmin(_ *gin.Context) bool { return true }

func (tp inviteToken) CreateVerificationToken(cfg *token.AuthTokenConfig, _ string, _ string) (string, error) {
	if tp.createVerificationToken != nil {
		return tp.createVerificationToken(cfg)
	}
	return "verification-token", nil
}

type inviteEmail struct{ email.Provider }

func (inviteEmail) SendEmail(_ []string, _ string, _ map[string]interface{}) error { return nil }

type inviteAudit struct{ audit.Provider }

func (inviteAudit) LogEvent(_ audit.Event) {}

func newInviteProvider(cfg *config.Config, st storage.Provider, tp token.Provider) *provider {
	log := zerolog.Nop()
	return &provider{
		Config: cfg,
		Dependencies: Dependencies{
			Log:             &log,
			StorageProvider: st,
			TokenProvider:   tp,
			EmailProvider:   inviteEmail{},
			AuditProvider:   inviteAudit{},
		},
	}
}

func inviteConfig() *config.Config {
	return &config.Config{
		IsEmailServiceEnabled:     true,
		EnableBasicAuthentication: true,
		EnableMagicLinkLogin:      true,
		DefaultRoles:              []string{"user"},
		AllowedOrigins:            []string{"*"},
	}
}

func inviteMeta() RequestMetadata {
	return RequestMetadata{Request: httptest.NewRequest(http.MethodPost, "http://example.com/", nil)}
}

func requireKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	require.Error(t, err)
	var se *Error
	require.Truef(t, errors.As(err, &se), "error must be a typed *service.Error, got %v", err)
	assert.Equal(t, want, se.Kind)
}

// TestInviteMembers_ReusesAddUserResult proves the redundant third loop was
// removed: GetUserByEmail is called exactly once per email (the existence
// pre-check), and every response user is sourced from AddUser's returned value.
func TestInviteMembers_ReusesAddUserResult(t *testing.T) {
	st := &inviteStorage{} // all emails new
	p := newInviteProvider(inviteConfig(), st, inviteToken{})

	emails := []string{"a@authorizer.dev", "b@authorizer.dev", "c@authorizer.dev"}
	res, _, err := p.InviteMembers(context.Background(), inviteMeta(), &model.InviteMemberRequest{Emails: emails})
	require.NoError(t, err)
	require.Len(t, res.Users, 3)

	assert.Equal(t, 3, st.getUserByEmailCalls,
		"GetUserByEmail must run once per email (existence check) — no redundant re-fetch after AddUser")
	assert.Equal(t, 3, st.addUserCalls)
	for _, u := range res.Users {
		// AddUser assigned ID = "id-"+email; a re-fetch path could not have.
		assert.Equal(t, "id-"+refs.StringValue(u.Email), u.ID)
	}
}

// TestInviteMembers_TokenErrorSkipsEmail proves the missing `continue` is fixed:
// a CreateVerificationToken failure on one email skips that email entirely
// (no AddUser, no invite) without corrupting or blocking the others.
func TestInviteMembers_TokenErrorSkipsEmail(t *testing.T) {
	st := &inviteStorage{}
	tp := inviteToken{createVerificationToken: func(cfg *token.AuthTokenConfig) (string, error) {
		if refs.StringValue(cfg.User.Email) == "bad@authorizer.dev" {
			return "", errors.New("token backend unavailable")
		}
		return "ok-token", nil
	}}
	p := newInviteProvider(inviteConfig(), st, tp)

	emails := []string{"good1@authorizer.dev", "bad@authorizer.dev", "good2@authorizer.dev"}
	res, _, err := p.InviteMembers(context.Background(), inviteMeta(), &model.InviteMemberRequest{Emails: emails})
	require.NoError(t, err, "one email's token failure must not fail the whole request")

	require.Len(t, res.Users, 2, "the failing email must be skipped, not invited with an empty token")
	assert.Equal(t, 2, st.addUserCalls, "AddUser must not run for the skipped email")
	assert.Equal(t, 2, st.addVerificationCalls)
	for _, u := range res.Users {
		assert.NotEqual(t, "bad@authorizer.dev", refs.StringValue(u.Email))
	}
}

// TestInviteMembers_RejectsOversizedBatch proves the batch cap: >100 emails is
// rejected as InvalidArgument before any storage work happens.
func TestInviteMembers_RejectsOversizedBatch(t *testing.T) {
	st := &inviteStorage{}
	p := newInviteProvider(inviteConfig(), st, inviteToken{})

	emails := make([]string, maxInviteMembersBatch+1)
	for i := range emails {
		emails[i] = fmt.Sprintf("u%d@authorizer.dev", i)
	}
	_, _, err := p.InviteMembers(context.Background(), inviteMeta(), &model.InviteMemberRequest{Emails: emails})

	requireKind(t, err, KindInvalidArgument)
	assert.Equal(t, 0, st.getUserByEmailCalls, "oversized batch must be rejected before touching storage")
	assert.Equal(t, 0, st.addUserCalls)
}

// TestAdminMethods_ValidationErrorsAreTyped covers representative admin methods
// (across three files) whose bare fmt.Errorf validation/conflict errors used to
// fall through to Internal/500. They must now carry the correct typed Kind so
// the transport maps them to 400 / 409.
func TestAdminMethods_ValidationErrorsAreTyped(t *testing.T) {
	ctx := context.Background()
	meta := inviteMeta()

	t.Run("CreateClient empty name -> InvalidArgument", func(t *testing.T) {
		p := newInviteProvider(inviteConfig(), &inviteStorage{}, inviteToken{})
		_, _, err := p.CreateClient(ctx, meta, &model.CreateClientRequest{Name: "   "})
		requireKind(t, err, KindInvalidArgument)
	})

	t.Run("AddTrustedIssuer missing service_account_id -> InvalidArgument", func(t *testing.T) {
		p := newInviteProvider(inviteConfig(), &inviteStorage{}, inviteToken{})
		_, _, err := p.AddTrustedIssuer(ctx, meta, &model.AddTrustedIssuerRequest{})
		requireKind(t, err, KindInvalidArgument)
	})

	t.Run("CreateOrganization duplicate name -> AlreadyExists", func(t *testing.T) {
		st := &inviteStorage{getOrgByName: func(name string) *schemas.Organization {
			return &schemas.Organization{ID: "org-1", Name: name}
		}}
		p := newInviteProvider(inviteConfig(), st, inviteToken{})
		_, _, err := p.CreateOrganization(ctx, meta, &model.CreateOrganizationRequest{Name: "acme"})
		requireKind(t, err, KindAlreadyExists)
	})
}
