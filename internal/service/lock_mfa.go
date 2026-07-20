// internal/service/lock_mfa.go
package service

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// LockMFA marks the user identified by the current MFA session as locked:
// they have no working path to complete MFA verification and must contact
// an admin. Permissions: none — identified via the MFA session cookie plus
// email/phone_number, same as SkipMFASetup/VerifyOTP.
func (p *provider) LockMFA(ctx context.Context, meta RequestMetadata, params *model.LockMfaRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "LockMFA").Logger()

	gc := &gin.Context{Request: meta.Request}
	mfaSession, err := cookie.GetMfaSession(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, nil, Unauthenticated(`invalid session`)
	}

	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))

	var user *schemas.User
	if email == "" && phoneNumber == "" {
		// No identifier supplied (OAuth-return MFA continuation): resolve the
		// account from the session cookie alone. Ownership plus a Verified
		// purpose prove the first factor, exactly as the GetMfaSession +
		// purpose check does on the identifier-supplied path below.
		ownerID, purpose, oErr := p.MemoryStoreProvider.GetMfaSessionOwner(mfaSession)
		if oErr != nil || purpose != constants.MFASessionPurposeVerified {
			log.Debug().Err(oErr).Msg("Failed to resolve mfa session owner")
			return nil, nil, Unauthenticated(`invalid session`)
		}
		user, err = p.StorageProvider.GetUserByID(ctx, ownerID)
		if user == nil || err != nil {
			log.Debug().Err(err).Msg("Failed to resolve user from mfa session")
			return nil, nil, Unauthenticated(`invalid session`)
		}
	} else {
		if email != "" {
			user, err = p.StorageProvider.GetUserByEmail(ctx, email)
		} else {
			user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		}
		if user == nil || err != nil {
			log.Debug().Err(err).Msg("User not found")
			return nil, nil, NotFound("invalid request")
		}

		// A bare session must not lock an account. Require a Verified session
		// (first factor completed for THIS user); a Challenge session
		// (ResendOTP/ForgotPassword) is rejected with the same shape as a missing
		// one, closing an unauthenticated account-lockout DoS.
		purpose, err := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession)
		if err != nil || purpose != constants.MFASessionPurposeVerified {
			log.Debug().Err(err).Msg("Failed to get mfa session")
			return nil, nil, Unauthenticated(`invalid session`)
		}
	}

	if p.hasVerifiedOTPFallback(ctx, user.ID) {
		log.Debug().Msg("User has a verified OTP fallback, refusing to lock")
		return nil, nil, FailedPrecondition("a verified email or SMS OTP fallback is available — use it instead of locking your account")
	}

	now := time.Now().Unix()
	user.MFALockedAt = &now
	if _, err := p.StorageProvider.UpdateUser(ctx, user); err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, nil, err
	}
	// Single-use: drop the session so a captured cookie cannot be replayed.
	_ = p.MemoryStoreProvider.DeleteMfaSession(user.ID, mfaSession)

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditMFALockedEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{Message: "Your account is locked. Contact your administrator to regain access."}, nil, nil
}

// hasVerifiedOTPFallback reports whether userID has a verified Email-OTP or
// SMS-OTP MFA enrollment (constants.EnvKeyEmailOTPAuthenticator /
// constants.EnvKeySMSOTPAuthenticator) — the one case where locking is
// refused because a working recovery path already exists and should be used
// instead.
func (p *provider) hasVerifiedOTPFallback(ctx context.Context, userID string) bool {
	for _, method := range []string{constants.EnvKeyEmailOTPAuthenticator, constants.EnvKeySMSOTPAuthenticator} {
		a, err := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, method)
		if err == nil && a != nil && a.VerifiedAt != nil {
			return true
		}
	}
	return false
}
