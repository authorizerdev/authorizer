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
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, nil, InvalidArgument(`email or phone number is required`)
	}
	isEmailVerification := email != ""
	isMobileVerification := phoneNumber != ""
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	// Get user by email or phone number
	var user *schemas.User
	if isEmailVerification {
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
	if _, err := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession); err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, nil, Unauthenticated(`invalid session`)
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
			log.Debug().Int64("attempts", attempts).Msg("TOTP verification locked: too many failed attempts")
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
		var otp *schemas.OTP
		if isEmailVerification {
			otp, err = p.StorageProvider.GetOTPByEmail(ctx, refs.StringValue(params.Email))
			if err != nil {
				log.Debug().Err(err).Msg("Failed to get otp request for email")
			}
		} else {
			otp, err = p.StorageProvider.GetOTPByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
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

	res, err := p.issueAuthResponse(ctx, meta, side, user, loginMethod, `OTP verified successfully.`, params.State, isSignUp)
	if err != nil {
		return nil, nil, err
	}
	return res, side, nil
}
