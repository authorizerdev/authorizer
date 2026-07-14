package service

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/authenticators/webauthn"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

// Dependencies are the subsystems a Provider needs. The set will grow as
// more operations migrate from internal/graphql into this package.
type Dependencies struct {
	Log *zerolog.Logger

	AuditProvider audit.Provider
	// AuthenticatorProvider registers and validates TOTP authenticators
	// (Google Authenticator) and recovery codes for MFA flows.
	AuthenticatorProvider authenticators.Provider
	// WebAuthnProvider runs WebAuthn/passkey registration and login ceremonies.
	WebAuthnProvider webauthn.Provider
	// AuthzEngine is the fine-grained authorization (FGA) engine.
	// It is nil unless an FGA store is configured (--fga-store);
	// FGA-gated operations MUST fail closed (return an error) when it is nil.
	AuthzEngine         engine.AuthorizationEngine
	EmailProvider       email.Provider
	EventsProvider      events.Provider
	MemoryStoreProvider memory_store.Provider
	SMSProvider         sms.Provider
	StorageProvider     storage.Provider
	TokenProvider       token.Provider
	// RateLimitProvider throttles abuse-prone admin ops (e.g. per-org domain
	// verification, which drives an outbound DNS lookup). Nil disables the limit.
	RateLimitProvider rate_limit.Provider
	// DNSResolver resolves TXT records for domain verification. Nil uses
	// net.DefaultResolver; tests inject a mock so no real DNS is hit.
	DNSResolver DNSResolver
}

// Provider is the transport-agnostic API for Authorizer public operations.
// Each method takes the inbound RequestMetadata and returns a typed response
// plus a ResponseSideEffects describing cookies (and other transport
// artifacts) the caller must apply.
//
// During the staged migration from internal/graphql, this interface grows one
// method per phase. Operations not yet migrated continue to live as
// graphqlProvider methods until they're moved here.
type Provider interface {
	// SignUp registers a new user. Public — no authentication required.
	SignUp(ctx context.Context, meta RequestMetadata, params *model.SignUpRequest) (*model.AuthResponse, *ResponseSideEffects, error)

	// Meta returns server discovery information (feature flags + provider
	// availability). Public — no authentication required.
	Meta(ctx context.Context, meta RequestMetadata) (*model.Meta, *ResponseSideEffects, error)

	// Profile returns the authenticated user. Requires session/bearer auth.
	Profile(ctx context.Context, meta RequestMetadata) (*model.User, *ResponseSideEffects, error)

	// CheckPermissions evaluates one or more fine-grained permission checks
	// for the caller (or, for super-admins, an explicit subject). Requires
	// session/bearer auth and a configured FGA engine (fail-closed).
	CheckPermissions(ctx context.Context, meta RequestMetadata, params *model.CheckPermissionsInput) (*model.CheckPermissionsResponse, *ResponseSideEffects, error)

	// ListPermissions enumerates what the caller (or, for super-admins, an
	// explicit subject) can access. Requires session/bearer auth and a
	// configured FGA engine (fail-closed).
	ListPermissions(ctx context.Context, meta RequestMetadata, params *model.ListPermissionsInput) (*model.ListPermissionsResponse, *ResponseSideEffects, error)

	// Logout ends the caller's current session. Browser callers get
	// expired Set-Cookie headers via side-effects. Requires auth.
	Logout(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error)

	// Revoke invalidates a refresh token. Typed mirror of RFC 7009.
	Revoke(ctx context.Context, meta RequestMetadata, params *model.OAuthRevokeRequest) (*model.Response, *ResponseSideEffects, error)

	// ValidateJwtToken validates a JWT (access/id/refresh) without rotation.
	ValidateJwtToken(ctx context.Context, meta RequestMetadata, params *model.ValidateJWTTokenRequest) (*model.ValidateJWTTokenResponse, *ResponseSideEffects, error)

	// ValidateSession validates a cookie session without rotation.
	ValidateSession(ctx context.Context, meta RequestMetadata, params *model.ValidateSessionRequest) (*model.ValidateSessionResponse, *ResponseSideEffects, error)

	// Session returns the AuthResponse bound to the caller's cookie/bearer
	// AND rotates the session token. Browser callers get a fresh
	// Set-Cookie via side-effects.
	Session(ctx context.Context, meta RequestMetadata, params *model.SessionQueryRequest) (*model.AuthResponse, *ResponseSideEffects, error)

	// DeactivateAccount marks the authenticated caller's account as revoked
	// and drops all of their sessions. Requires auth.
	DeactivateAccount(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error)

	// SkipMFASetup completes a token-withheld first-time MFA offer by
	// recording the decline and issuing the previously-withheld token.
	// Identified via the MFA session cookie, not a bearer token — none
	// exists yet at this point in the flow.
	SkipMFASetup(ctx context.Context, meta RequestMetadata, params *model.SkipMfaSetupRequest) (*model.AuthResponse, *ResponseSideEffects, error)

	// LockMFA records that the authenticated-in-progress caller lost access
	// to their only MFA factor(s). Requires no verified Email/SMS OTP
	// fallback exists for the user — otherwise that should be used instead.
	// Does not issue a token.
	LockMFA(ctx context.Context, meta RequestMetadata, params *model.LockMfaRequest) (*model.Response, *ResponseSideEffects, error)

	// EmailOTPMFASetup sends a one-time code to the caller's own email and
	// begins an email-OTP MFA enrollment. Verified via VerifyOTP. Requires
	// an authenticated caller (bearer token) — a settings-screen action.
	EmailOTPMFASetup(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error)
	// SMSOTPMFASetup is EmailOTPMFASetup's SMS twin.
	SMSOTPMFASetup(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error)

	// ResendVerifyEmail re-issues a pending email-verification link. Public —
	// response is generic to avoid account enumeration.
	ResendVerifyEmail(ctx context.Context, meta RequestMetadata, params *model.ResendVerifyEmailRequest) (*model.Response, *ResponseSideEffects, error)

	// ResendOTP re-issues a one-time passcode for an MFA/verification
	// challenge. Public.
	ResendOTP(ctx context.Context, meta RequestMetadata, params *model.ResendOTPRequest) (*model.Response, *ResponseSideEffects, error)

	// ForgotPassword issues a password-reset token (email) or OTP (SMS).
	// Public — response is generic to avoid account enumeration.
	ForgotPassword(ctx context.Context, meta RequestMetadata, params *model.ForgotPasswordRequest) (*model.ForgotPasswordResponse, *ResponseSideEffects, error)

	// ResetPassword completes a password reset using a verification token
	// (email) or OTP (SMS). Public.
	ResetPassword(ctx context.Context, meta RequestMetadata, params *model.ResetPasswordRequest) (*model.Response, *ResponseSideEffects, error)

	// UpdateProfile updates the authenticated caller's profile. Requires auth.
	// May rotate/clear the session cookie (e.g. on email change) via
	// side-effects.
	UpdateProfile(ctx context.Context, meta RequestMetadata, params *model.UpdateProfileRequest) (*model.Response, *ResponseSideEffects, error)

	// MagicLinkLogin sends a passwordless login link. Public — response is
	// generic to avoid account enumeration.
	MagicLinkLogin(ctx context.Context, meta RequestMetadata, params *model.MagicLinkLoginRequest) (*model.Response, *ResponseSideEffects, error)

	// Login authenticates a user via email/phone + password, issuing tokens
	// or initiating an MFA challenge. Browser callers get Set-Cookie via
	// side-effects. Public.
	Login(ctx context.Context, meta RequestMetadata, params *model.LoginRequest) (*model.AuthResponse, *ResponseSideEffects, error)

	// VerifyEmail completes email verification and logs the user in. Browser
	// callers get a session cookie via side-effects. Public.
	VerifyEmail(ctx context.Context, meta RequestMetadata, params *model.VerifyEmailRequest) (*model.AuthResponse, *ResponseSideEffects, error)

	// VerifyOTP validates an email/SMS OTP or TOTP/recovery code and logs the
	// user in. Browser callers get a session cookie via side-effects. Public.
	VerifyOTP(ctx context.Context, meta RequestMetadata, params *model.VerifyOTPRequest) (*model.AuthResponse, *ResponseSideEffects, error)

	// WebauthnRegistrationOptions begins a passkey registration ceremony for the
	// authenticated caller. Requires a session. Public (self-service).
	WebauthnRegistrationOptions(ctx context.Context, meta RequestMetadata, email *string) (*model.WebauthnRegistrationOptionsResponse, error)
	// WebauthnRegistrationVerify verifies the attestation and stores the passkey
	// for the authenticated caller. Requires a session. Public (self-service).
	WebauthnRegistrationVerify(ctx context.Context, meta RequestMetadata, params *model.WebauthnRegistrationVerifyRequest) (*model.Response, error)
	// WebauthnLoginOptions begins a passkey login ceremony — usernameless when
	// email is nil, else scoped to that user's credentials. Public.
	WebauthnLoginOptions(ctx context.Context, meta RequestMetadata, email *string) (*model.WebauthnLoginOptionsResponse, error)
	// WebauthnLoginVerify verifies a passkey assertion and logs the user in.
	// Browser callers get a session cookie via side-effects. Public.
	WebauthnLoginVerify(ctx context.Context, meta RequestMetadata, params *model.WebauthnLoginVerifyRequest) (*model.AuthResponse, *ResponseSideEffects, error)
	// WebauthnCredentials lists the authenticated caller's own passkeys. Requires
	// a session. Public (self-service).
	WebauthnCredentials(ctx context.Context, meta RequestMetadata) ([]*model.WebauthnCredentialInfo, error)
	// WebauthnDeleteCredential deletes one of the authenticated caller's own
	// passkeys. Requires a session. Public (self-service).
	WebauthnDeleteCredential(ctx context.Context, meta RequestMetadata, id string) (*model.Response, error)
}

// New constructs a new service provider.
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	return &provider{
		Config:       cfg,
		Dependencies: *deps,
	}, nil
}

type provider struct {
	*config.Config
	Dependencies
}

// Compile-time check that provider satisfies Provider.
var _ Provider = (*provider)(nil)
