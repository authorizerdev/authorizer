package http_handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// accountStateStore is a storage.Provider stub covering only the lookups
// accountHasState performs.
type accountStateStore struct {
	storage.Provider
	memberships      []*schemas.OrgMembership
	credentials      []*schemas.WebauthnCredential
	authenticator    *schemas.Authenticator
	membershipErr    error
	credentialErr    error
	authenticatorErr error
}

func (s *accountStateStore) ListOrgMembershipsByUser(_ context.Context, _ string, _ *model.Pagination) ([]*schemas.OrgMembership, *model.Pagination, error) {
	return s.memberships, nil, s.membershipErr
}

func (s *accountStateStore) ListWebauthnCredentialsByUserID(_ context.Context, _ string) ([]*schemas.WebauthnCredential, error) {
	return s.credentials, s.credentialErr
}

func (s *accountStateStore) GetAuthenticatorDetailsByUserId(_ context.Context, _ string, _ string) (*schemas.Authenticator, error) {
	if s.authenticatorErr != nil {
		return nil, s.authenticatorErr
	}
	if s.authenticator == nil {
		// gorm's own sentinel, not errors.New("not found"): accountHasState has
		// to tell "not enrolled" (the normal case, per the not-found contract)
		// apart from "the lookup failed", and a bare error string is
		// indistinguishable from a database fault.
		return nil, gorm.ErrRecordNotFound
	}
	return s.authenticator, nil
}

// stubAuthzEngine reports a fixed set of tuples for any ReadTuples call.
type stubAuthzEngine struct {
	engine.AuthorizationEngine
	tuples []engine.TupleKey
	err    error
}

func (e *stubAuthzEngine) ReadTuples(_ context.Context, _ engine.ReadTuplesFilter) (*engine.ReadTuplesResult, error) {
	if e.err != nil {
		return nil, e.err
	}
	return &engine.ReadTuplesResult{Tuples: e.tuples}, nil
}

// TestAccountHasState guards the bound on the pre-hijack delete.
//
// That guard deletes an unverified pre-existing account rather than linking a
// federated identity to it. StorageProvider.DeleteUser cascades to sessions and
// nothing else, and the replacement account gets a fresh id — so deleting an
// account that holds anything means its FGA grants, org memberships and
// enrolled authenticators are silently lost, and any federated-identity row is
// left pointing at a dead user id, hard-locking that principal out of SSO.
//
// Only a genuinely empty account (a real squatter) may be deleted.
func TestAccountHasState(t *testing.T) {
	t.Parallel()

	newProvider := func(store storage.Provider, authz engine.AuthorizationEngine) *httpProvider {
		logger := zerolog.Nop()
		return &httpProvider{
			Config: &config.Config{},
			Dependencies: Dependencies{
				Log:             &logger,
				StorageProvider: store,
				AuthzEngine:     authz,
			},
		}
	}
	user := &schemas.User{ID: "user-1"}

	t.Run("an empty account is deletable", func(t *testing.T) {
		t.Parallel()
		h := newProvider(&accountStateStore{}, nil)
		hasState, what := h.accountHasState(context.Background(), user)
		assert.False(t, hasState, "a never-used squatter account holds nothing")
		assert.Empty(t, what)
	})

	t.Run("org membership blocks deletion", func(t *testing.T) {
		t.Parallel()
		h := newProvider(&accountStateStore{
			memberships: []*schemas.OrgMembership{{OrgID: "org-1", UserID: "user-1"}},
		}, nil)
		hasState, what := h.accountHasState(context.Background(), user)
		assert.True(t, hasState)
		assert.Equal(t, "org membership", what)
	})

	t.Run("a passkey blocks deletion", func(t *testing.T) {
		t.Parallel()
		h := newProvider(&accountStateStore{
			credentials: []*schemas.WebauthnCredential{{UserID: "user-1"}},
		}, nil)
		hasState, what := h.accountHasState(context.Background(), user)
		assert.True(t, hasState)
		assert.Equal(t, "passkey", what)
	})

	t.Run("an enrolled authenticator blocks deletion", func(t *testing.T) {
		t.Parallel()
		h := newProvider(&accountStateStore{
			authenticator: &schemas.Authenticator{UserID: "user-1"},
		}, nil)
		hasState, what := h.accountHasState(context.Background(), user)
		assert.True(t, hasState)
		assert.Equal(t, "enrolled authenticator", what)
	})

	t.Run("an FGA grant blocks deletion", func(t *testing.T) {
		t.Parallel()
		// The permissions nothing would ever clean up, and the ones the user
		// most visibly loses.
		h := newProvider(&accountStateStore{}, &stubAuthzEngine{
			tuples: []engine.TupleKey{{User: "user:user-1", Relation: "viewer", Object: "document:1"}},
		})
		hasState, what := h.accountHasState(context.Background(), user)
		assert.True(t, hasState)
		assert.Equal(t, "fga grant", what)
	})

	t.Run("a lookup fault reports state rather than allowing deletion", func(t *testing.T) {
		t.Parallel()
		// When we cannot tell, refusing the login is recoverable and deleting
		// an account is not.
		for _, tc := range []struct {
			name  string
			store *accountStateStore
			authz engine.AuthorizationEngine
			want  string
		}{
			{"membership fault", &accountStateStore{membershipErr: errors.New("db down")}, nil, "org membership lookup failed"},
			{"passkey fault", &accountStateStore{credentialErr: errors.New("db down")}, nil, "passkey lookup failed"},
			// Not the same as "not enrolled": a fault here used to be swallowed
			// (`a, _ :=`), so a database blip looked like an empty account and
			// deleted somebody's MFA enrollment.
			{"authenticator fault", &accountStateStore{authenticatorErr: errors.New("db down")}, nil, "authenticator lookup failed"},
			{"fga fault", &accountStateStore{}, &stubAuthzEngine{err: errors.New("fga down")}, "fga tuple lookup failed"},
		} {
			h := newProvider(tc.store, tc.authz)
			hasState, what := h.accountHasState(context.Background(), user)
			assert.True(t, hasState, tc.name)
			assert.Equal(t, tc.want, what, tc.name)
		}
	})
}
