// Package scim implements a per-organization inbound SCIM 2.0 server for user
// provisioning and deprovisioning (RFC 7643/7644, users only). A customer's IdP
// (Okta, Entra, …) authenticates with a per-org bearer token and the org it may
// act on is derived ONLY from the matched endpoint — never from the URL or
// payload (design §4.4 H6, the C3 confused-deputy class).
//
// ponytail: users only. SCIM Groups → org-namespaced FGA roles is deferred to a
// follow-up (needs the FGA engine + org-admin permission model, design §4.4
// CR2). Also deferred: ETag/versioning and pagination cursors.
package scim

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Sentinel errors the transport maps to SCIM status codes.
var (
	// ErrUnauthorized — missing/malformed/invalid bearer token, or a disabled
	// endpoint. Map to 401. Deliberately generic: it never distinguishes an
	// unknown endpoint id from a wrong secret (constant-time, dummy-compare).
	ErrUnauthorized = errors.New("scim: unauthorized")
	// ErrNotFound — the resource does not exist OR belongs to another org (H6:
	// a cross-org id is indistinguishable from a non-existent one). Map to 404.
	ErrNotFound = errors.New("scim: resource not found")
	// ErrConflict — a create collides with an existing userName owned outside
	// this org (global email uniqueness). Map to 409.
	ErrConflict = errors.New("scim: userName already exists")
	// ErrInvalid — malformed input (e.g. missing userName). Map to 400.
	ErrInvalid = errors.New("scim: invalid request")
)

// tokenCost is the bcrypt cost for SCIM bearer-token secrets. Matches the
// client-secret cost so a dummy compare is timing-indistinguishable.
const tokenCost = 12

var (
	dummyHash []byte
	dummyOnce sync.Once
)

// performDummyCompare burns an equivalent bcrypt cost for an unknown endpoint so
// timing does not reveal whether an endpoint id exists (mirrors clientauth).
func performDummyCompare(secret string) {
	dummyOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-scim-token"), tokenCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(secret))
}

// User is the transport-neutral SCIM user projection the handler maps to/from
// SCIM JSON. Only the attributes Okta/Entra send for provisioning are modelled.
type User struct {
	ExternalID string
	UserName   string // SCIM userName → the user's email
	GivenName  string
	FamilyName string
	Active     bool
}

// Dependencies for the SCIM service.
type Dependencies struct {
	Log                 *zerolog.Logger
	StorageProvider     storage.Provider
	MemoryStoreProvider memory_store.Provider
}

// Provider is the org-bounded SCIM operation surface. Every method takes the
// orgID resolved by Authenticate — callers MUST NOT source it from the request
// path or body (H6).
type Provider interface {
	// Authenticate verifies the presented bearer token and returns the org it
	// authorizes. The org is derived solely from the matched endpoint.
	Authenticate(ctx context.Context, bearer string) (orgID string, err error)

	// CreateUser provisions a user into the org. Idempotent: a repeat with the
	// same externalId (or an existing org member with the same userName) returns
	// the existing user with existed=true and creates no duplicate.
	CreateUser(ctx context.Context, orgID string, in User) (user *schemas.User, existed bool, err error)
	// GetUser fetches an org member by id (404 if not a member — H6).
	GetUser(ctx context.Context, orgID, userID string) (*schemas.User, error)
	// FindByUserName returns the org member with the given userName, or nil when
	// none (the IdP's pre-create dedup probe). Never leaks another org's user.
	FindByUserName(ctx context.Context, orgID, userName string) (*schemas.User, error)
	// ReplaceUser (PUT) overwrites the mutable profile + active flag of an org
	// member. A true→false active transition revokes the user's sessions.
	ReplaceUser(ctx context.Context, orgID, userID string, in User) (*schemas.User, error)
	// SetActive (PATCH active / DELETE) flips the active flag. Deactivation
	// synchronously revokes the user's sessions + refresh tokens.
	SetActive(ctx context.Context, orgID, userID string, active bool) (*schemas.User, error)
}

type provider struct {
	Dependencies
}

var _ Provider = &provider{}

// New constructs a SCIM service provider.
func New(deps *Dependencies) Provider {
	return &provider{Dependencies: *deps}
}

// namespacedExternalID composes the org-scoped external id key. Storing and
// looking up external ids in this form is what makes GetUserByExternalID
// org-isolating without a cross-table join (H6, works identically on all 6 DBs).
func namespacedExternalID(orgID, externalID string) string {
	return orgID + ":" + externalID
}

// Authenticate parses "<endpointID>.<hexSecret>", resolves the endpoint by id,
// and constant-time verifies the secret against its bcrypt hash.
func (p *provider) Authenticate(ctx context.Context, bearer string) (string, error) {
	log := p.Log.With().Str("func", "scim.Authenticate").Logger()
	id, secret, ok := strings.Cut(strings.TrimSpace(bearer), ".")
	if !ok || id == "" || secret == "" {
		performDummyCompare(bearer)
		return "", ErrUnauthorized
	}
	endpoint, err := p.StorageProvider.GetScimEndpointByID(ctx, id)
	if err != nil || endpoint == nil {
		// Unknown endpoint id: burn an equivalent bcrypt cost so an unknown id
		// and a wrong secret take the same time.
		log.Debug().Msg("scim endpoint not found for presented token")
		performDummyCompare(secret)
		return "", ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(endpoint.TokenHash), []byte(secret)) != nil {
		log.Debug().Str("org_id", endpoint.OrgID).Msg("scim token secret mismatch")
		return "", ErrUnauthorized
	}
	if !endpoint.Enabled {
		log.Debug().Str("org_id", endpoint.OrgID).Msg("scim endpoint disabled")
		return "", ErrUnauthorized
	}
	return endpoint.OrgID, nil
}

// GenerateToken builds "<endpointID>.<hexSecret>" with 256 bits of entropy and
// returns the plaintext plus its bcrypt hash. Only the hash is persisted; the
// plaintext is revealed once at create/rotate. Stateless — the admin surface
// calls it directly when provisioning an endpoint.
func GenerateToken(endpointID string) (plaintext, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	secret := hex.EncodeToString(raw)
	h, err := bcrypt.GenerateFromPassword([]byte(secret), tokenCost)
	if err != nil {
		return "", "", err
	}
	return endpointID + "." + secret, string(h), nil
}

// requireMember is the H6 isolation gate: a user is visible/mutable through a
// SCIM connection only if they hold a membership in that connection's org. A
// cross-org id therefore returns ErrNotFound, never another org's data.
func (p *provider) requireMember(ctx context.Context, orgID, userID string) (*schemas.User, error) {
	if _, err := p.StorageProvider.GetOrgMembership(ctx, orgID, userID); err != nil {
		return nil, ErrNotFound
	}
	user, err := p.StorageProvider.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

func (p *provider) CreateUser(ctx context.Context, orgID string, in User) (*schemas.User, bool, error) {
	log := p.Log.With().Str("func", "scim.CreateUser").Str("org_id", orgID).Logger()
	userName := strings.TrimSpace(in.UserName)
	if userName == "" {
		return nil, false, ErrInvalid
	}

	// Dedup #1: same externalId already provisioned into this org (org-scoped
	// composite key). Idempotent — return the existing user.
	if in.ExternalID != "" {
		if existing, err := p.StorageProvider.GetUserByExternalID(ctx, orgID, in.ExternalID); err == nil && existing != nil {
			log.Debug().Msg("dedup by external_id")
			return existing, true, nil
		}
	}

	// Dedup #2: a user with this userName (email) already exists. If they are a
	// member of this org, return them (idempotent). If they exist but belong to
	// another org, email is globally unique so we cannot re-provision — 409.
	if existing, err := p.StorageProvider.GetUserByEmail(ctx, userName); err == nil && existing != nil {
		if _, mErr := p.StorageProvider.GetOrgMembership(ctx, orgID, existing.ID); mErr == nil {
			log.Debug().Msg("dedup by userName within org")
			return existing, true, nil
		}
		// ponytail: accepted risk. Authorizer enforces global email uniqueness, so
		// a userName already owned by another org cannot be re-provisioned here —
		// return 409. This leaks only that *some* account with that email exists
		// (an existence oracle on an email the caller's IdP already knows), never
		// the other org's user data or membership. H6 (by-id read/mutate isolation)
		// is unaffected. Upgrade path if even existence must be hidden: per-org
		// user rows keyed by (org, email) instead of global email uniqueness.
		log.Debug().Msg("userName exists outside this org")
		return nil, false, ErrConflict
	}

	now := time.Now().Unix()
	email := userName
	nsExt := namespacedExternalID(orgID, in.ExternalID)
	newUser := &schemas.User{
		ID:            uuid.New().String(),
		Email:         &email,
		SignupMethods: "scim",
		IsActive:      in.Active,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if in.ExternalID != "" {
		newUser.ExternalID = &nsExt
	}
	if in.GivenName != "" {
		gn := in.GivenName
		newUser.GivenName = &gn
	}
	if in.FamilyName != "" {
		fn := in.FamilyName
		newUser.FamilyName = &fn
	}
	// A user provisioned as active:false is deprovisioned from birth — stamp the
	// revocation marker so no token can ever be minted for them.
	if !in.Active {
		newUser.RevokedTimestamp = &now
	}
	created, err := p.StorageProvider.AddUser(ctx, newUser)
	if err != nil {
		log.Debug().Err(err).Msg("failed to add scim user")
		return nil, false, err
	}
	// GORM's `default:true` on IsActive means a Create with IsActive=false is
	// persisted as true (Go zero-value → column default). Force it via a Save
	// so a user provisioned directly as inactive is actually stored inactive.
	// (RevokedTimestamp above already blocks token issuance regardless.)
	if !in.Active {
		created.IsActive = false
		updated, uErr := p.StorageProvider.UpdateUser(ctx, created)
		if uErr != nil {
			log.Debug().Err(uErr).Msg("failed to persist inactive state on provisioned user")
			return nil, false, uErr
		}
		created = updated
	}
	// Bind the user to the org. Without this membership the user would not be an
	// org member and every subsequent by-id op would (correctly) 404.
	if _, err := p.StorageProvider.AddOrgMembership(ctx, &schemas.OrgMembership{
		OrgID:  orgID,
		UserID: created.ID,
		Roles:  "",
	}); err != nil {
		log.Debug().Err(err).Msg("failed to add org membership for scim user")
		return nil, false, err
	}
	return created, false, nil
}

func (p *provider) GetUser(ctx context.Context, orgID, userID string) (*schemas.User, error) {
	return p.requireMember(ctx, orgID, userID)
}

func (p *provider) FindByUserName(ctx context.Context, orgID, userName string) (*schemas.User, error) {
	userName = strings.TrimSpace(userName)
	if userName == "" {
		return nil, nil
	}
	user, err := p.StorageProvider.GetUserByEmail(ctx, userName)
	if err != nil || user == nil {
		return nil, nil
	}
	// Only surface the user if they belong to this org (H6): otherwise the probe
	// would confirm another org's user exists.
	if _, err := p.StorageProvider.GetOrgMembership(ctx, orgID, user.ID); err != nil {
		return nil, nil
	}
	return user, nil
}

func (p *provider) ReplaceUser(ctx context.Context, orgID, userID string, in User) (*schemas.User, error) {
	user, err := p.requireMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	wasActive := user.IsActive
	if in.GivenName != "" {
		gn := in.GivenName
		user.GivenName = &gn
	}
	if in.FamilyName != "" {
		fn := in.FamilyName
		user.FamilyName = &fn
	}
	user.IsActive = in.Active
	if wasActive && !in.Active {
		return p.deactivate(ctx, user)
	}
	if !wasActive && in.Active {
		user.RevokedTimestamp = nil
	}
	user.UpdatedAt = time.Now().Unix()
	return p.StorageProvider.UpdateUser(ctx, user)
}

func (p *provider) SetActive(ctx context.Context, orgID, userID string, active bool) (*schemas.User, error) {
	user, err := p.requireMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	if !active {
		if !user.IsActive {
			return user, nil
		}
		user.IsActive = false
		return p.deactivate(ctx, user)
	}
	// Reactivation: clear the revocation marker.
	user.IsActive = true
	user.RevokedTimestamp = nil
	user.UpdatedAt = time.Now().Unix()
	return p.StorageProvider.UpdateUser(ctx, user)
}

// deactivate is the enterprise-offboarding path: mark the user inactive+revoked
// and SYNCHRONOUSLY drop every session and refresh/session token so a held
// access token stops working immediately (across instances via the shared
// store). This is the whole point of SCIM deprovisioning.
func (p *provider) deactivate(ctx context.Context, user *schemas.User) (*schemas.User, error) {
	log := p.Log.With().Str("func", "scim.deactivate").Str("user_id", user.ID).Logger()
	now := time.Now().Unix()
	user.IsActive = false
	user.RevokedTimestamp = &now
	user.UpdatedAt = now
	updated, err := p.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("failed to update user on deactivate")
		return nil, err
	}
	// Kill live sessions/access/refresh tokens in the shared memory store
	// (validation for all three token types routes through GetUserSession,
	// keyed by user id) and any DB-backed session tokens. Synchronous — the
	// caller must not observe a still-valid token after deprovision returns.
	if err := p.MemoryStoreProvider.DeleteAllUserSessions(updated.ID); err != nil {
		log.Debug().Err(err).Msg("failed to delete user sessions from memory store")
	}
	if err := p.StorageProvider.DeleteAllSessionTokensByUserID(ctx, updated.ID); err != nil {
		log.Debug().Err(err).Msg("failed to delete session tokens from storage")
	}
	return updated, nil
}
