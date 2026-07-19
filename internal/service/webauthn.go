package service

import (
	"context"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/gin-gonic/gin"
)

// resolveWebauthnSetupCaller resolves who is calling
// WebauthnRegistrationOptions/WebauthnRegistrationVerify. The ordinary
// settings-page caller is bearer-token authenticated (unchanged). During a
// token-withheld MFA offer (mfaGateOfferAll/mfaGateBlockEnroll — see
// mfa_gate.go) there is no bearer token yet, only the MFA session cookie, so
// registering a passkey there authenticates the same way VerifyOTP/
// SkipMFASetup do: the cookie's Verified purpose proves the first factor
// already completed for this exact user. The gate is recomputed and must
// still be a genuine enrollment offer — never mfaGateBlockVerify, or a caller
// who only proved a password could mint a brand-new passkey and skip
// challenging their EXISTING second factor, defeating it entirely.
func (p *provider) resolveWebauthnSetupCaller(ctx context.Context, meta RequestMetadata, email, phoneNumber string) (*schemas.User, bool, error) {
	if tokenData, tErr := p.callerTokenData(ctx, meta); tErr == nil && tokenData != nil && tokenData.UserID != "" {
		user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
		if err != nil {
			return nil, false, err
		}
		return user, false, nil
	}

	gc := &gin.Context{Request: meta.Request}
	mfaSession, err := cookie.GetMfaSession(gc)
	if err != nil {
		return nil, false, Unauthenticated("unauthorized")
	}

	email = strings.TrimSpace(email)
	phoneNumber = strings.TrimSpace(phoneNumber)
	var user *schemas.User
	if email == "" && phoneNumber == "" {
		ownerID, purpose, oErr := p.MemoryStoreProvider.GetMfaSessionOwner(mfaSession)
		if oErr != nil || purpose != constants.MFASessionPurposeVerified {
			return nil, false, Unauthenticated("invalid session")
		}
		user, err = p.StorageProvider.GetUserByID(ctx, ownerID)
	} else if email != "" {
		user, err = p.StorageProvider.GetUserByEmail(ctx, email)
	} else {
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
	}
	if user == nil || err != nil {
		return nil, false, Unauthenticated("invalid session")
	}
	if email != "" || phoneNumber != "" {
		purpose, pErr := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession)
		if pErr != nil || purpose != constants.MFASessionPurposeVerified {
			return nil, false, Unauthenticated("invalid session")
		}
	}

	gate := resolveMFAGate(
		effectiveMFAEnabled(p.Config, user),
		p.Config.EnforceMFA,
		p.authenticatorVerified(ctx, user.ID),
		user.HasSkippedMFASetupAt != nil,
	)
	if gate != mfaGateOfferAll && gate != mfaGateBlockEnroll {
		return nil, false, FailedPrecondition("cannot set up a passkey in the current state")
	}
	return user, true, nil
}

// WebauthnRegistrationOptions begins a passkey registration ceremony for the
// caller — either bearer-token authenticated (settings page) or MFA-session
// authenticated (login-time enrollment offer); see resolveWebauthnSetupCaller.
//
// Permissions: authenticated:user, or an MFA-session-cookie caller mid-offer.
func (p *provider) WebauthnRegistrationOptions(ctx context.Context, meta RequestMetadata, email, phoneNumber *string) (*model.WebauthnRegistrationOptionsResponse, error) {
	log := p.Log.With().Str("func", "WebauthnRegistrationOptions").Logger()
	user, _, err := p.resolveWebauthnSetupCaller(ctx, meta, refs.StringValue(email), refs.StringValue(phoneNumber))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve caller")
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
// persists the passkey for the caller (see resolveWebauthnSetupCaller). When
// resolved via the MFA session (login-time enrollment offer, not the
// settings page), this also completes the MFA gate and issues the
// previously-withheld auth token — exactly like totp_mfa_setup +
// verify_otp(is_totp: true) does for TOTP.
//
// Permissions: authenticated:user, or an MFA-session-cookie caller mid-offer.
func (p *provider) WebauthnRegistrationVerify(ctx context.Context, meta RequestMetadata, params *model.WebauthnRegistrationVerifyRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "WebauthnRegistrationVerify").Logger()
	side := &ResponseSideEffects{}
	user, sessionAuthenticated, err := p.resolveWebauthnSetupCaller(ctx, meta, refs.StringValue(params.Email), refs.StringValue(params.PhoneNumber))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, nil, err
	}
	name := ""
	if params.Name != nil {
		name = strings.TrimSpace(refs.StringValue(params.Name))
	}
	cred, err := p.WebAuthnProvider.FinishRegistration(ctx, meta.HostURL, user, name, params.Credential)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to finish registration")
		return nil, nil, InvalidArgument(err.Error())
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
	if !sessionAuthenticated {
		return &model.AuthResponse{Message: "Passkey registered successfully."}, side, nil
	}
	// Single-use: drop the session so a captured cookie cannot be replayed.
	gc := &gin.Context{Request: meta.Request}
	if mfaSession, sErr := cookie.GetMfaSession(gc); sErr == nil {
		_ = p.MemoryStoreProvider.DeleteMfaSession(user.ID, mfaSession)
	}
	res, err := p.issueAuthResponse(ctx, meta, side, user, constants.AuthRecipeMethodWebauthn, "Passkey registered and MFA setup complete.", params.State, false)
	if err != nil {
		return nil, nil, err
	}
	return res, side, nil
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

	if user.MFALockedAt != nil {
		log.Debug().Msg("User's MFA is locked, refusing passkey login")
		return nil, nil, FailedPrecondition("your account's multi-factor authentication is locked; contact your administrator to regain access")
	}

	// A successful WebAuthn assertion satisfies the MFA requirement on its own,
	// full stop — whether this is the user's primary/first login action or an
	// explicitly-offered second factor after a password login, and regardless of
	// what other factors (TOTP, email-OTP, SMS-OTP) are also enrolled. This
	// deployment registers passkeys with UserVerification: Required (see the
	// webauthn provider), so every ceremony already bundles device possession
	// with a local biometric/PIN; it is treated as sufficient the same way
	// verify_otp issues a token once a TOTP/OTP code validates. There is no
	// further gate and no OTP/TOTP re-challenge: reaching this point (past the
	// revoked/email-verified/MFA-locked guards above) issues the token directly.
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
