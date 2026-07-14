package service

import (
	"context"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/gin-gonic/gin"
)

// WebauthnRegistrationOptions begins a passkey registration ceremony for the
// authenticated caller. The session — not the email argument — identifies the
// user, so a caller can only register a passkey against their own account.
//
// Permissions: authenticated:user
func (p *provider) WebauthnRegistrationOptions(ctx context.Context, meta RequestMetadata, email *string) (*model.WebauthnRegistrationOptionsResponse, error) {
	log := p.Log.With().Str("func", "WebauthnRegistrationOptions").Logger()
	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil || tokenData == nil || tokenData.UserID == "" {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, Unauthenticated("unauthorized")
	}
	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, err
	}
	options, err := p.WebAuthnProvider.BeginRegistration(ctx, meta.HostURL, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to begin registration")
		return nil, InvalidArgument("failed to start passkey registration")
	}
	return &model.WebauthnRegistrationOptionsResponse{Options: options}, nil
}

// WebauthnRegistrationVerify verifies the attestation from the browser and
// persists the passkey for the authenticated caller.
//
// Permissions: authenticated:user
func (p *provider) WebauthnRegistrationVerify(ctx context.Context, meta RequestMetadata, params *model.WebauthnRegistrationVerifyRequest) (*model.Response, error) {
	log := p.Log.With().Str("func", "WebauthnRegistrationVerify").Logger()
	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil || tokenData == nil || tokenData.UserID == "" {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, Unauthenticated("unauthorized")
	}
	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, err
	}
	name := ""
	if params.Name != nil {
		name = strings.TrimSpace(refs.StringValue(params.Name))
	}
	cred, err := p.WebAuthnProvider.FinishRegistration(ctx, meta.HostURL, user, name, params.Credential)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to finish registration")
		return nil, InvalidArgument(err.Error())
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditWebauthnCredentialAddedEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   cred.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "Passkey registered successfully."}, nil
}

// WebauthnLoginOptions begins a passkey login ceremony. With no email it is a
// usernameless (discoverable) ceremony that surfaces any resident passkey; with
// an email it is scoped to that user's own credentials (MFA-alternative flow).
//
// Permissions: none — this begins an authentication.
func (p *provider) WebauthnLoginOptions(ctx context.Context, meta RequestMetadata, email *string) (*model.WebauthnLoginOptionsResponse, error) {
	log := p.Log.With().Str("func", "WebauthnLoginOptions").Logger()
	emailStr := strings.TrimSpace(refs.StringValue(email))
	if emailStr == "" {
		options, err := p.WebAuthnProvider.BeginDiscoverableLogin(ctx, meta.HostURL)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to begin discoverable login")
			return nil, InvalidArgument("failed to start passkey login")
		}
		return &model.WebauthnLoginOptionsResponse{Options: options}, nil
	}
	// Scoped (MFA-alternative) flow. This must only ever run for a caller who
	// already completed password authentication for THIS SPECIFIC account -
	// otherwise a client-supplied email lets an unauthenticated caller probe
	// "does this account have a passkey?" one-shot (the real
	// PublicKeyCredentialRequestOptions returned on success, including that
	// account's own credential IDs in allowCredentials, is itself the leak).
	// The MFA session cookie is exactly the proof-of-password-auth verify_otp
	// already requires for the equivalent TOTP-alternative flow, so we gate
	// on it the same way here.
	gc := &gin.Context{Request: meta.Request}
	mfaSession, mfaErr := cookie.GetMfaSession(gc)
	if mfaErr != nil {
		log.Debug().Err(mfaErr).Msg("Failed to get mfa session")
		return nil, Unauthenticated(`invalid session`)
	}
	user, err := p.StorageProvider.GetUserByEmail(ctx, emailStr)
	if err != nil || user == nil {
		log.Debug().Err(err).Msg("User not found for scoped webauthn login")
		return nil, NotFound("no passkey found for this account")
	}
	if _, err := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession); err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, Unauthenticated(`invalid session`)
	}
	options, err := p.WebAuthnProvider.BeginLogin(ctx, meta.HostURL, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to begin scoped login")
		return nil, NotFound("no passkey found for this account")
	}
	return &model.WebauthnLoginOptionsResponse{Options: options}, nil
}

// WebauthnLoginVerify verifies a passkey assertion and logs the user in. It is
// intentionally STRICTER than password login: the account's email MUST be
// verified before a passkey can issue tokens (a locked design decision), and it
// returns a distinct error when it is not so the frontend can prompt the user
// to verify their email rather than reporting an invalid credential.
//
// Permissions: none — completes an authentication.
func (p *provider) WebauthnLoginVerify(ctx context.Context, meta RequestMetadata, params *model.WebauthnLoginVerifyRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "WebauthnLoginVerify").Logger()
	side := &ResponseSideEffects{}

	// The provider resolves the owning user (usernameless) or verifies against
	// the pinned user (scoped) and validates the signature.
	user, _, err := p.WebAuthnProvider.FinishLogin(ctx, meta.HostURL, params.Credential)
	if err != nil || user == nil {
		log.Debug().Err(err).Msg("Failed to verify passkey assertion")
		return nil, nil, Unauthenticated("invalid passkey")
	}

	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, nil, FailedPrecondition("user access has been revoked")
	}

	// Locked policy: a passkey may not issue tokens until the account's email is
	// verified. Distinct, actionable error — never the generic invalid-credential.
	if user.EmailVerifiedAt == nil {
		log.Debug().Msg("Email not verified — refusing passkey login")
		return nil, nil, FailedPrecondition("email is not verified. please verify your email before signing in with a passkey")
	}

	// A passkey used for PRIMARY login is only one factor (something you
	// have) — it does not itself satisfy an MFA requirement, so it goes
	// through the exact same 5-way gate password login does. A WebAuthn
	// credential registered for MFA purposes on this same account (there is
	// no `purpose` field distinguishing "primary" vs "MFA" registrations)
	// counts as a verified second factor here too, same as login.go's TOTP
	// branch treats it — but the credential the user just authenticated
	// PRIMARY with cannot also be counted as its own second factor, so
	// authenticatorVerified below is TOTP-only for a passkey-primary login.
	authenticator, authErr := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
	totpVerified := authErr == nil && authenticator != nil && authenticator.VerifiedAt != nil
	gate := resolveMFAGate(
		effectiveMFAEnabled(p.Config, user),
		p.Config.EnforceMFA,
		totpVerified,
		user.HasSkippedMFASetupAt != nil,
	)
	switch gate {
	case mfaGateBlockVerify:
		if !p.Config.EnableTOTPLogin {
			log.Debug().Msg("EnforceMFA is on but no compatible second factor is configured for passkey login")
			return nil, nil, FailedPrecondition("multi-factor authentication is required but no compatible verification method is available for passkey sign-in; please sign in with your password instead")
		}
		expiresAt := time.Now().Add(3 * time.Minute).Unix()
		if err := p.setMFASession(meta, side, user.ID, expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, nil, err
		}
		return &model.AuthResponse{
			Message:              `Proceed to mfa verification`,
			ShouldShowTotpScreen: refs.NewBoolRef(true),
		}, side, nil
	case mfaGateBlockEnroll:
		if !p.Config.EnableTOTPLogin {
			log.Debug().Msg("EnforceMFA is on but no compatible second factor is configured for passkey login")
			return nil, nil, FailedPrecondition("multi-factor authentication is required but no compatible verification method is available for passkey sign-in; please sign in with your password instead")
		}
		expiresAt := time.Now().Add(3 * time.Minute).Unix()
		if err := p.setMFASession(meta, side, user.ID, expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, nil, err
		}
		enrollment, err := p.generateTOTPEnrollment(ctx, user.ID)
		if err != nil {
			log.Debug().Msg("Failed to generate totp")
			return nil, nil, err
		}
		return &model.AuthResponse{
			Message:                    `Proceed to totp verification screen`,
			ShouldShowTotpScreen:       refs.NewBoolRef(true),
			AuthenticatorScannerImage:  refs.NewStringRef(enrollment.ScannerImage),
			AuthenticatorSecret:        refs.NewStringRef(enrollment.Secret),
			AuthenticatorRecoveryCodes: enrollment.RecoveryCodes,
		}, side, nil
	case mfaGateOfferAll:
		expiresAt := time.Now().Add(3 * time.Minute).Unix()
		if err := p.setMFASession(meta, side, user.ID, expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, nil, err
		}
		res := &model.AuthResponse{
			Message:                     `Proceed to mfa setup`,
			ShouldOfferWebauthnMfaSetup: refs.NewBoolRef(p.Config.EnableWebauthnMFA),
		}
		// Unlike login.go's TOTP branch (only reachable when EnableTOTPLogin is
		// already true), passkey-primary login reaches this gate regardless of
		// TOTP availability — only offer/generate a TOTP enrollment when TOTP
		// login is actually enabled server-wide, or p.AuthenticatorProvider is
		// nil and generateTOTPEnrollment panics. The token is withheld either
		// way via setMFASession above; WebAuthn-only offer is still meaningful
		// when TOTP isn't configured.
		if p.Config.EnableTOTPLogin {
			enrollment, err := p.generateTOTPEnrollment(ctx, user.ID)
			if err != nil {
				log.Debug().Msg("Failed to generate totp for optional setup")
				return nil, nil, err
			}
			res.ShouldShowTotpScreen = refs.NewBoolRef(true)
			res.AuthenticatorScannerImage = refs.NewStringRef(enrollment.ScannerImage)
			res.AuthenticatorSecret = refs.NewStringRef(enrollment.Secret)
			res.AuthenticatorRecoveryCodes = enrollment.RecoveryCodes
		}
		return res, side, nil
	case mfaGateSkippedSetup, mfaGateNone:
		// Both fall through to normal token issuance below.
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditLoginSuccessEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	res, err := p.issueAuthResponse(ctx, meta, side, user, constants.AuthRecipeMethodWebauthn, "Logged in successfully with passkey.", params.State, false)
	if err != nil {
		return nil, nil, err
	}
	return res, side, nil
}

// WebauthnCredentials lists the authenticated caller's own passkeys.
//
// Permissions: authenticated:user
func (p *provider) WebauthnCredentials(ctx context.Context, meta RequestMetadata) ([]*model.WebauthnCredentialInfo, error) {
	log := p.Log.With().Str("func", "WebauthnCredentials").Logger()
	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil || tokenData == nil || tokenData.UserID == "" {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, Unauthenticated("unauthorized")
	}
	creds, err := p.StorageProvider.ListWebauthnCredentialsByUserID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to list credentials")
		return nil, err
	}
	out := make([]*model.WebauthnCredentialInfo, 0, len(creds))
	for _, c := range creds {
		out = append(out, c.AsAPIWebauthnCredential())
	}
	return out, nil
}

// WebauthnDeleteCredential deletes one of the authenticated caller's own
// passkeys. It authorizes against the session's user id — never a client
// supplied one — and reports a not-found error when the credential is missing
// OR owned by another user, so it can't be used to probe for other users'
// credentials.
//
// Permissions: authenticated:user
func (p *provider) WebauthnDeleteCredential(ctx context.Context, meta RequestMetadata, id string) (*model.Response, error) {
	log := p.Log.With().Str("func", "WebauthnDeleteCredential").Logger()
	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil || tokenData == nil || tokenData.UserID == "" {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, Unauthenticated("unauthorized")
	}
	cred, err := p.StorageProvider.GetWebauthnCredentialByID(ctx, id)
	if err != nil || cred == nil || cred.UserID != tokenData.UserID {
		log.Debug().Err(err).Msg("Credential not found or not owned by caller")
		return nil, NotFound("passkey not found")
	}
	if err := p.StorageProvider.DeleteWebauthnCredential(ctx, cred); err != nil {
		log.Debug().Err(err).Msg("Failed to delete credential")
		return nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditWebauthnCredentialDeletedEvent,
		Protocol: meta.Protocol, ActorID: tokenData.UserID,
		ActorType:    constants.AuditActorTypeUser,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   cred.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "Passkey deleted successfully."}, nil
}
