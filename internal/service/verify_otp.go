package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const (
	// totpMaxFailedAttempts is the number of failed TOTP/recovery-code
	// verifications tolerated for a single user before verification is
	// locked. The global per-IP rate limiter does not stop an attacker
	// spreading brute-force guesses for one account across many IPs, so we
	// additionally cap failures per user.
	totpMaxFailedAttempts = 5
	// totpLockoutWindowSeconds is both the sliding window over which
	// failures accumulate and the duration verification stays locked once
	// the threshold is hit (15 minutes).
	totpLockoutWindowSeconds = 15 * 60
	// totpLockoutCachePrefix namespaces the per-user failed-attempt counter
	// in the memory store. This is transient state, deliberately kept out
	// of the DB schema so no storage provider needs a new column.
	totpLockoutCachePrefix = "totp_failed_attempts:"
	// otpLockoutCachePrefix namespaces the per-user email/SMS OTP
	// failed-attempt counter, mirroring totpLockoutCachePrefix. The email/SMS
	// OTP branch reuses the TOTP threshold/window policy constants above
	// (totpMaxFailedAttempts / totpLockoutWindowSeconds) — same brute-force
	// exposure past the per-IP limiter, same mitigation — but on its own cache
	// key so the two factors' counters never collide.
	otpLockoutCachePrefix = "otp_failed_attempts:"
)

// VerifyOTP verifies a one-time passcode (email/SMS OTP, TOTP, or recovery
// code) for a pending MFA session and, on success, issues an auth token.
// Transport-agnostic port of graphqlProvider.VerifyOTP.
//
// Permissions: none — completes an in-progress authentication identified by the
// MFA session cookie.
func (p *provider) VerifyOTP(ctx context.Context, meta RequestMetadata, params *model.VerifyOTPRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "VerifyOTP").Logger()
	side := &ResponseSideEffects{}

	// The MFA session lives in a request cookie; cookie.GetMfaSession still
	// reads from a gin.Context, so wrap the inbound *http.Request.
	gc := &gin.Context{Request: meta.Request}
	mfaSession, err := cookie.GetMfaSession(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, nil, Unauthenticated(`invalid session`)
	}

	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	isEmailVerification := email != ""
	isMobileVerification := phoneNumber != ""
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	// Get user by email or phone number
	var user *schemas.User
	// sessionResolved is true when the caller supplied no identifier and the
	// account was resolved from the MFA session cookie alone (OAuth-return MFA
	// continuation, where the frontend never learns the account's email/phone).
	// Session ownership is then already proven by GetMfaSessionOwner, so the
	// later GetMfaSession re-check is skipped for this path.
	sessionResolved := false
	if email == "" && phoneNumber == "" {
		// No identifier supplied (OAuth-return MFA continuation): only a
		// Verified session may resolve an account this way — every legitimate
		// no-identifier caller (oauth_mfa_gate.go, resend_otp.go's session-only
		// fallback) already upgrades to Verified before reaching here.
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
		email = strings.TrimSpace(refs.StringValue(user.Email))
		phoneNumber = strings.TrimSpace(refs.StringValue(user.PhoneNumber))
		isEmailVerification = email != ""
		isMobileVerification = !isEmailVerification && phoneNumber != ""
		sessionResolved = true
	} else if isEmailVerification {
		user, err = p.StorageProvider.GetUserByEmail(ctx, refs.StringValue(params.Email))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
		}
	} else {
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
		}
	}
	if user == nil || err != nil {
		log.Debug().Err(err).Msg("User not found")
		return nil, nil, NotFound("invalid verification request")
	}
	// Rebuild the logger from scratch (not derived from the email/phone_number
	// context above) now that user.ID is known: every remaining log line in
	// this function — including the lockout Warn below, which is enabled by
	// default in production unlike the Debug lines above — must not carry raw
	// PII. user_id is an opaque internal identifier, not personal data.
	log = p.Log.With().Str("func", "VerifyOTP").Str("user_id", user.ID).Logger()

	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, nil, FailedPrecondition("invalid verification request")
	}

	// Validate the MFA session before doing ANY code/lockout-counter work.
	// This must run first: the TOTP lockout counter below is keyed by
	// user.ID alone, resolvable from a bare email with no proof the caller
	// ever completed the password step. Checking the session first means an
	// attacker who only knows a victim's email - with no valid session - is
	// rejected before they can touch (and so exhaust) the victim's lockout
	// counter, closing an unauthenticated account-lockout DoS.
	if !sessionResolved {
		// Verified (real login/signup/OAuth MFA challenge) and Challenge
		// (ResendOTP's identifier-supplied hand-off) are both legitimate here.
		// PasswordReset is deliberately excluded: a forgot-password OTP session
		// must only ever be redeemable through ResetPassword, never through
		// VerifyOTP for a token — see constants.MFASessionPurposePasswordReset.
		purpose, err := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession)
		if err != nil || (purpose != constants.MFASessionPurposeVerified && purpose != constants.MFASessionPurposeChallenge) {
			log.Debug().Err(err).Msg("Failed to get mfa session")
			return nil, nil, Unauthenticated(`invalid session`)
		}
	}

	// Verify OTP based on TOPT or OTP
	if refs.BoolValue(params.IsTotp) {
		// Per-user lockout: atomically reserve this attempt's slot BEFORE
		// validating, then check whether it exceeded the budget. This is
		// deliberately increment-then-check rather than check-then-increment:
		// under concurrent requests, IncrementCache still hands out strictly
		// increasing, unique counts (1,2,3,...), so at most
		// totpMaxFailedAttempts requests can ever reach validation in a
		// window no matter how many arrive simultaneously. A
		// check-then-increment design (read count, compare, validate, then
		// write count+1) lets arbitrarily many concurrent requests all read
		// the same pre-increment count and all pass the check, defeating the
		// lockout entirely - parallelizing the exact brute-force attack this
		// exists to stop.
		lockKey := totpLockoutCachePrefix + user.ID
		attempts, incErr := p.MemoryStoreProvider.IncrementCache(lockKey, totpLockoutWindowSeconds)
		if incErr != nil {
			// A memory-store fault must not be counted as a user failure or
			// block a legitimate user (avoids self-lockout on outage) - same
			// fail-open philosophy already used below for
			// ValidateRecoveryCode's storage-fault case.
			log.Debug().Err(incErr).Msg("Failed to increment totp failed-attempt counter")
		} else if attempts > totpMaxFailedAttempts {
			metrics.RecordSecurityEvent("totp_verification_locked", "verify_otp")
			log.Warn().Int64("attempts", attempts).Str("ip", meta.IPAddress).Msg("TOTP verification locked: too many failed attempts")
			return nil, nil, TooManyRequests(`too many failed attempts, please try again later`)
		}

		verified, err := p.AuthenticatorProvider.Validate(ctx, params.Otp, user.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to validate passcode")
			return nil, nil, errors.New("error while validating passcode")
		}
		if !verified {
			log.Info().Msg("TOTP passcode invalid, checking if it is a recovery code")
			// ValidateRecoveryCode returns (false, nil) for an invalid or
			// already-used code; a non-nil error is a storage fault and must
			// not be counted as a user failure (avoids self-lockout on outage).
			verified, err = p.AuthenticatorProvider.ValidateRecoveryCode(ctx, params.Otp, user.ID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to validate recovery code")
				return nil, nil, errors.New("error while validating recovery code")
			}
		}
		if !verified {
			log.Debug().Msg("Failed to verify otp request: Incorrect value")
			return nil, nil, InvalidArgument(`invalid otp`)
		}
		// Successful verification clears the failed-attempt counter for this user.
		if cErr := p.MemoryStoreProvider.DeleteCacheByPrefix(lockKey); cErr != nil {
			log.Debug().Err(cErr).Msg("Failed to reset totp failed-attempt counter")
		}
	} else {
		// Per-user lockout for email/SMS OTP, identical in shape to the TOTP
		// branch above: atomically reserve this attempt's slot BEFORE
		// validating, then check whether it exceeded the budget. Increment-
		// then-check (not check-then-increment) so concurrent requests get
		// strictly increasing unique counts and at most totpMaxFailedAttempts
		// can ever reach validation in a window. The MFA session was already
		// validated above, so an attacker who only knows the victim's email
		// cannot reach here to exhaust this counter (no unauthenticated
		// account-lockout DoS).
		otpLockKey := otpLockoutCachePrefix + user.ID
		attempts, incErr := p.MemoryStoreProvider.IncrementCache(otpLockKey, totpLockoutWindowSeconds)
		if incErr != nil {
			// A memory-store fault must not be counted as a user failure or
			// lock out a legitimate user (fail-open, matching the TOTP branch).
			log.Debug().Err(incErr).Msg("Failed to increment otp failed-attempt counter")
		} else if attempts > totpMaxFailedAttempts {
			metrics.RecordSecurityEvent("otp_verification_locked", "verify_otp")
			log.Warn().Int64("attempts", attempts).Str("ip", meta.IPAddress).Msg("OTP verification locked: too many failed attempts")
			return nil, nil, TooManyRequests(`too many failed attempts, please try again later`)
		}

		var otp *schemas.OTP
		if isEmailVerification {
			otp, err = p.StorageProvider.GetOTPByEmail(ctx, email)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to get otp request for email")
			}
		} else {
			otp, err = p.StorageProvider.GetOTPByPhoneNumber(ctx, phoneNumber)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to get otp request for phone number")
			}
		}
		if otp == nil {
			log.Debug().Msg("OTP not found")
			return nil, nil, NotFound(`OTP not found`)
		}
		// OTPs are stored as HMAC-SHA256 digests so an offline DB dump no
		// longer reveals usable codes. We deliberately do NOT fall back
		// to literal equality — accepting the stored value verbatim
		// would turn the digest itself into a usable credential.
		if !crypto.VerifyOTPHash(params.Otp, otp.Otp, p.Config.JWTSecret) {
			log.Debug().Msg("Failed to verify otp request: OTP mismatch")
			return nil, nil, InvalidArgument(`invalid otp`)
		}
		expiresIn := otp.ExpiresAt - time.Now().Unix()
		if expiresIn < 0 {
			log.Debug().Msg("OTP expired")
			return nil, nil, InvalidArgument("otp expired")
		}
		// Successful verification clears the failed-attempt counter for this user.
		if cErr := p.MemoryStoreProvider.DeleteCacheByPrefix(otpLockKey); cErr != nil {
			log.Debug().Err(cErr).Msg("Failed to reset otp failed-attempt counter")
		}
		if err := p.StorageProvider.DeleteOTP(ctx, otp); err != nil {
			log.Debug().Err(err).Msg("Failed to delete otp")
		}

		// Mark the corresponding email/SMS-OTP MFA enrollment verified, but
		// ONLY when a pending (unverified) Authenticator row already exists
		// for this method — i.e. the caller went through
		// EmailOTPMFASetup/SMSOTPMFASetup first. A plain login-time OTP
		// send/verify (login.go's pre-enrollment challenge, or a signup
		// email/phone verification) never created that row, so this is a
		// no-op for those: routine login-time OTP must not silently
		// "enroll" anyone as MFA.
		method := constants.EnvKeyEmailOTPAuthenticator
		if isMobileVerification {
			method = constants.EnvKeySMSOTPAuthenticator
		}
		if authenticator, aErr := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, method); aErr == nil && authenticator != nil && authenticator.VerifiedAt == nil {
			now := time.Now().Unix()
			authenticator.VerifiedAt = &now
			if _, err := p.StorageProvider.UpdateAuthenticator(ctx, authenticator); err != nil {
				log.Debug().Err(err).Msg("Failed to mark otp authenticator verified")
			}
		}
	}

	isSignUp := false
	if user.EmailVerifiedAt == nil && isEmailVerification {
		isSignUp = true
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}
	if user.PhoneNumberVerifiedAt == nil && isMobileVerification {
		isSignUp = true
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}
	if isSignUp {
		user, err = p.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to update user")
			return nil, nil, err
		}
	}
	loginMethod := constants.AuthRecipeMethodBasicAuth
	if isMobileVerification {
		loginMethod = constants.AuthRecipeMethodMobileOTP
	}
	if isEmailVerification {
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditEmailVerifiedEvent,
			Protocol: meta.Protocol, ActorID: user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
	} else {
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditPhoneVerifiedEvent,
			Protocol: meta.Protocol, ActorID: user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
	}

	// Single-use: the OTP is verified, so drop the session to prevent replay of
	// a captured cookie within its remaining TTL.
	_ = p.MemoryStoreProvider.DeleteMfaSession(user.ID, mfaSession)

	res, err := p.issueAuthResponse(ctx, meta, side, user, loginMethod, `OTP verified successfully.`, params.State, isSignUp)
	if err != nil {
		return nil, nil, err
	}
	return res, side, nil
}
