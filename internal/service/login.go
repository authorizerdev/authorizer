package service

import (
	"strings"
	"sync"
	"time"

	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/codestate"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// loginGenericErrMsg is the single error returned for any authentication
// failure. All distinct failure causes (user not found, wrong password, email
// not verified, wrong auth method, account revoked) collapse to this message to
// prevent user enumeration. The real reason is recorded in the debug log for
// ops visibility.
const loginGenericErrMsg = "invalid credentials"

const (
	// loginMaxFailedAttempts / loginLockoutWindowSeconds bound how many wrong
	// passwords a single account will accept before the password compare stops
	// running at all. The global per-IP rate limiter does not help here: an
	// attacker guessing one victim's password rotates source IPs (an IPv6 /64,
	// a botnet, open proxies) and faces no per-account cap at all. Values match
	// the TOTP/OTP lockout (verify_otp.go) so a user sees one consistent policy
	// across every credential.
	//
	// The trade-off this makes, stated plainly rather than wished away: ANY
	// per-account lockout is a denial of service against that account, and this
	// one is keyed on user id but reachable by anyone who knows the email. Six
	// wrong guesses lock the account, and because IncrementCache refreshes the
	// TTL on every attempt the window slides — an attacker spending one request
	// every few minutes keeps a victim locked out indefinitely. The window is
	// kept short so a real user recovers quickly on their own, and it is the
	// same policy verify_otp.go applies, so a user sees one consistent rule
	// across every credential. Removing the DoS entirely needs something this
	// lockout is not (progressive delay, or throttling the source rather than
	// the account); until then, blocking the password brute force is judged the
	// better of the two exposures.
	loginMaxFailedAttempts    = 5
	loginLockoutWindowSeconds = 15 * 60
	// loginLockoutCachePrefix namespaces the per-user counter, keyed by user
	// id (never by email) so the counter itself cannot be used to probe which
	// addresses are registered.
	loginLockoutCachePrefix = "login_failed_attempts:"
)

// mfaScopeKey namespaces the requested scope carried across an MFA
// interruption, keyed by the MFA session id so it expires with it.
func mfaScopeKey(mfaSession string) string {
	return "mfa_scope:" + mfaSession
}

// consumeMFAScope returns the scope stashed by setMFASession and clears it, so
// a captured cookie cannot replay the scope after the session is spent. A
// missing or unreadable entry yields nil, and the caller falls back to the
// default scope — losing scope must never block a legitimate login.
func (p *provider) consumeMFAScope(mfaSession string) []string {
	if mfaSession == "" {
		return nil
	}
	key := mfaScopeKey(mfaSession)
	blob, err := p.MemoryStoreProvider.GetCache(key)
	if err != nil || blob == "" {
		return nil
	}
	_ = p.MemoryStoreProvider.DeleteCacheByPrefix(key)
	var scope []string
	if err := json.Unmarshal([]byte(blob), &scope); err != nil {
		return nil
	}
	return scope
}

// setMFASession arms a short-lived MFA session (memory-store entry + cookie)
// proving the caller already completed a first authentication factor for
// userID. verify_otp and the scoped webauthn_login_options/_verify flow both
// require this session before they'll act. Shared by Login's TOTP branch and
// WebauthnLoginVerify's EnforceMFA gate.
//
// The optional scope is the caller's requested OAuth scope, carried across
// the interruption so the deferred token issuance can honour it.
func (p *provider) setMFASession(meta RequestMetadata, side *ResponseSideEffects, userID string, expiresAt int64, scope ...string) error {
	mfaSession := uuid.NewString()
	// Every caller of this helper (login, webauthn-verify, oauth callback) has
	// already confirmed a first factor for userID, so the session is Verified —
	// the only purpose skip_mfa_setup/lock_mfa will act on.
	if err := p.MemoryStoreProvider.SetMfaSession(userID, mfaSession, constants.MFASessionPurposeVerified, expiresAt); err != nil {
		return err
	}
	// Carry the caller's requested scope across the MFA interruption.
	//
	// The token is issued later, by skip_mfa_setup / verify_otp / webauthn —
	// none of which receive the original request. Without this the issuance
	// path fell back to a hardcoded ["openid","email","profile"], silently
	// dropping every other scope the caller asked for. Delegation flows lost
	// exactly the scopes they exist to attenuate.
	//
	// Deliberately NOT re-supplied by the client at completion: those
	// endpoints are unauthenticated, so accepting a scope there would let a
	// caller self-grant privileges they never requested at login. It is
	// carried, never re-asked.
	//
	// Held in the cache rather than on the MFA session row: the row has no
	// scope column and adding one means a migration across all six storage
	// backends, for state that is transient and already expires with the
	// session. Same pattern as the pending-TOTP-secret and OTP-lockout keys.
	if len(scope) > 0 {
		ttl := expiresAt - time.Now().Unix()
		if ttl > 0 {
			if blob, err := json.Marshal(scope); err == nil {
				if err := p.MemoryStoreProvider.SetCache(mfaScopeKey(mfaSession), string(blob), ttl); err != nil {
					// Non-fatal: losing the scope degrades to the default set
					// rather than blocking a legitimate login.
					p.Log.Debug().Err(err).Msg("failed to persist mfa session scope")
				}
			}
		}
	}
	for _, c := range cookie.BuildMfaSessionCookies(meta.HostURL, mfaSession, p.Config.AppCookieSecure, expiresAt) {
		side.AddCookie(c)
	}
	return nil
}

// loginDummyBcryptHash is a precomputed bcrypt hash used to equalise the
// response time of the user-not-found path with the real password verification
// path. Without this, an attacker can distinguish "no such user" from "wrong
// password" by measuring response latency (no bcrypt vs one bcrypt).
var (
	loginDummyBcryptHash []byte
	loginDummyBcryptOnce sync.Once
)

// loginPerformDummyPasswordCheck runs a constant-cost bcrypt comparison whose
// result is intentionally discarded. Call it on the user-not-found / no-password
// branches so the request still does roughly the same amount of CPU work as a
// real authentication attempt.
func loginPerformDummyPasswordCheck(password string) {
	loginDummyBcryptOnce.Do(func() {
		// generated lazily so cost depends on bcrypt.DefaultCost at runtime
		loginDummyBcryptHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(loginDummyBcryptHash, []byte(password))
}

// totpEnrollment is a freshly generated (unverified) TOTP enrollment
// payload, shared by both the mfaGateBlockEnroll (forced) and
// mfaGateOfferAll (optional) paths of the TOTP MFA branch below.
type totpEnrollment struct {
	ScannerImage  string
	Secret        string
	RecoveryCodes []*string
}

// generateTOTPEnrollment generates a new TOTP secret/QR/recovery-codes for
// userID. Extracted so the TOTP MFA branch doesn't duplicate this call across
// its "block until enrolled" and "offer setup" cases.
func (p *provider) generateTOTPEnrollment(ctx context.Context, userID string) (*totpEnrollment, error) {
	authConfig, err := p.AuthenticatorProvider.Generate(ctx, userID)
	if err != nil {
		return nil, err
	}
	recoveryCodes := []*string{}
	for _, code := range authConfig.RecoveryCodes {
		recoveryCodes = append(recoveryCodes, refs.NewStringRef(code))
	}
	return &totpEnrollment{
		ScannerImage:  authConfig.ScannerImage,
		Secret:        authConfig.Secret,
		RecoveryCodes: recoveryCodes,
	}, nil
}

// Login authenticates a user with email or phone number (not both).
// Transport-agnostic port of graphqlProvider.Login.
//
// Permissions: none.
func (p *provider) Login(ctx context.Context, meta RequestMetadata, params *model.LoginRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Login").Logger()
	side := &ResponseSideEffects{}

	isBasicAuthEnabled := p.Config.EnableBasicAuthentication
	isMobileBasicAuthEnabled := p.Config.EnableMobileBasicAuthentication
	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, nil, InvalidArgument(`email or phone number is required`)
	}
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if !isBasicAuthEnabled && isEmailLogin {
		log.Debug().Msg("Basic authentication is disabled")
		return nil, nil, FailedPrecondition(`basic authentication is disabled for this instance`)
	}
	if !isMobileBasicAuthEnabled && isMobileLogin {
		log.Debug().Msg("Mobile basic authentication is disabled")
		return nil, nil, FailedPrecondition(`mobile basic authentication is disabled for this instance`)
	}
	var user *schemas.User
	var err error
	if isEmailLogin {
		user, err = p.StorageProvider.GetUserByEmail(ctx, email)
		if err == nil {
			log.Debug().Str("email", email).Msg("User found by email")
		}
	} else {
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err == nil {
			log.Debug().Str("phone_number", phoneNumber).Msg("User found by phone number")
		}
	}
	if err != nil {
		log.Debug().Err(err).Str("reason", "user_not_found").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		// Equalise response timing with the real bcrypt path so an attacker
		// cannot distinguish "no such user" from "wrong password" by latency.
		loginPerformDummyPasswordCheck(params.Password)
		return nil, nil, Unauthenticated(loginGenericErrMsg)
	}
	if user.RevokedTimestamp != nil {
		log.Debug().Str("reason", "account_revoked").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		metrics.RecordSecurityEvent("account_revoked", "login_attempt")
		loginPerformDummyPasswordCheck(params.Password)
		return nil, nil, Unauthenticated(loginGenericErrMsg)
	}
	isEmailServiceEnabled := p.Config.IsEmailServiceEnabled
	isSMSServiceEnabled := p.Config.IsSMSServiceEnabled
	if isEmailLogin {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodBasicAuth) {
			log.Debug().Str("reason", "wrong_signup_method").Msg("login failed")
			loginPerformDummyPasswordCheck(params.Password)
			return nil, nil, Unauthenticated(loginGenericErrMsg)
		}

		if user.EmailVerifiedAt == nil {
			// Check if email service is enabled
			// Send email verification via otp
			if !isEmailServiceEnabled {
				log.Debug().Str("reason", "email_not_verified_no_email_service").Msg("login failed")
				loginPerformDummyPasswordCheck(params.Password)
				return nil, nil, Unauthenticated(loginGenericErrMsg)
			} else {
				if vreq, err := p.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup); err == nil && vreq != nil {
					// if verification request exists and not expired then return
					// if verification request exists and expired then delete it and proceed
					if vreq.ExpiresAt > time.Now().Unix() {
						log.Debug().Str("reason", "email_verification_pending").Msg("login failed")
						loginPerformDummyPasswordCheck(params.Password)
						return nil, nil, Unauthenticated(loginGenericErrMsg)
					} else {
						if err := p.StorageProvider.DeleteVerificationRequest(ctx, vreq); err != nil {
							log.Debug().Msg("Failed to delete verification request")
							return nil, nil, err
						} else {
							log.Debug().Msg("Verification request deleted")
						}
					}
				}
				expiresAt := time.Now().Add(1 * time.Minute).Unix()
				otpData, err := p.generateAndStoreOTP(ctx, user, expiresAt)
				if err != nil {
					log.Debug().Msg("Failed to generate otp")
					return nil, nil, err
				}
				if err := p.setMFASession(meta, side, user.ID, expiresAt, params.Scope...); err != nil {
					log.Debug().Msg("Failed to set mfa session")
					return nil, nil, err
				}
				asyncutil.Go(p.Log, func() {
					ctx := context.WithoutCancel(ctx)
					// exec it as go routine so that we can reduce the api latency
					if err := p.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]any{
						"user":         user.ToMap(),
						"organization": utils.GetOrganization(p.Config),
						"otp":          otpData.Otp,
					}); err != nil {
						log.Debug().Msg("Failed to send otp email")
					}
					_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
				})
				return &model.AuthResponse{
					Message:                  "Please check email inbox for the OTP",
					ShouldShowEmailOtpScreen: refs.NewBoolRef(isEmailLogin),
				}, side, nil
			}
		}
	} else {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth) {
			log.Debug().Str("reason", "wrong_signup_method_phone").Msg("login failed")
			loginPerformDummyPasswordCheck(params.Password)
			return nil, nil, Unauthenticated(loginGenericErrMsg)
		}

		if user.PhoneNumberVerifiedAt == nil {
			if !isSMSServiceEnabled {
				log.Debug().Str("reason", "phone_not_verified_no_sms_service").Msg("login failed")
				loginPerformDummyPasswordCheck(params.Password)
				return nil, nil, Unauthenticated(loginGenericErrMsg)
			} else {
				expiresAt := time.Now().Add(1 * time.Minute).Unix()
				otpData, err := p.generateAndStoreOTP(ctx, user, expiresAt)
				if err != nil {
					log.Debug().Msg("Failed to generate otp")
					return nil, nil, err
				}
				if err := p.setMFASession(meta, side, user.ID, expiresAt, params.Scope...); err != nil {
					log.Debug().Msg("Failed to set mfa session")
					return nil, nil, err
				}
				asyncutil.Go(p.Log, func() {
					ctx := context.WithoutCancel(ctx)
					smsBody := strings.Builder{}
					smsBody.WriteString("Your verification code is: ")
					smsBody.WriteString(otpData.Otp)
					_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
					if err := p.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
						log.Debug().Msg("Failed to send sms")
					}
				})
				return &model.AuthResponse{
					Message:                   "Please check text message for the OTP",
					ShouldShowMobileOtpScreen: refs.NewBoolRef(isMobileLogin),
				}, side, nil
			}
		}
	}
	// Per-account lockout: atomically reserve this attempt's slot BEFORE the
	// password compare, then check whether it exceeded the budget. Deliberately
	// increment-then-check, exactly as verify_otp.go documents: IncrementCache
	// hands out strictly increasing unique counts under concurrency, so at most
	// loginMaxFailedAttempts requests can ever reach the compare in a window no
	// matter how many arrive at once. Check-then-increment would let arbitrarily
	// many parallel requests read the same pre-increment count and all pass,
	// parallelizing the brute force this exists to stop.
	loginLockKey := loginLockoutCachePrefix + user.ID
	if attempts, incErr := p.MemoryStoreProvider.IncrementCache(loginLockKey, loginLockoutWindowSeconds); incErr != nil {
		// A memory-store fault must not be counted as a user failure or lock a
		// legitimate user out during an outage — same fail-open stance as the
		// OTP path.
		log.Debug().Err(incErr).Msg("Failed to increment login failed-attempt counter")
	} else if attempts > loginMaxFailedAttempts {
		metrics.RecordSecurityEvent("login_locked", "login")
		log.Warn().Int64("attempts", attempts).Str("ip", meta.IPAddress).Msg("Login locked: too many failed attempts")
		return nil, nil, TooManyRequests(`too many failed attempts, please try again later`)
	}
	if user.Password == nil {
		// A basic_auth user with no stored hash (e.g. a pre-fix Couchbase
		// record that never persisted one) must fail the same way as a
		// wrong password, not nil-pointer-dereference.
		loginPerformDummyPasswordCheck(params.Password)
		err = bcrypt.ErrMismatchedHashAndPassword
	} else {
		err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(params.Password))
	}
	if err != nil {
		log.Debug().Str("reason", "bad_password").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		metrics.RecordSecurityEvent("invalid_credentials", "bad_password")
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditLoginFailedEvent,
			Protocol: meta.Protocol, ActorID: user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
		return nil, nil, Unauthenticated(loginGenericErrMsg)
	}
	// The password is proved: clear the counter so a legitimate user who
	// mistyped a few times starts fresh. Any MFA factor below carries its own
	// independent lockout.
	if cErr := p.MemoryStoreProvider.DeleteCacheByPrefix(loginLockKey); cErr != nil {
		log.Debug().Err(cErr).Msg("Failed to reset login failed-attempt counter")
	}
	roles := p.Config.DefaultRoles
	currentRoles := strings.Split(user.Roles, ",")
	if len(params.Roles) > 0 {
		if !validators.IsValidRoles(params.Roles, currentRoles) {
			log.Debug().Msg("Invalid roles")
			return nil, nil, InvalidArgument(`invalid roles`)
		}
		roles = params.Roles
	}
	scope := []string{"openid", "email", "profile"}
	if len(params.Scope) > 0 {
		scope = params.Scope
	}

	isMFAEnabled := p.Config.EnableMFA
	isTOTPLoginEnabled := p.Config.EnableTOTPLogin
	isMailOTPEnabled := p.Config.EnableEmailOTP
	isSMSOTPEnabled := p.Config.EnableSMSOTP

	// A single check protecting all three MFA branches below (email-OTP,
	// SMS-OTP, TOTP/resolveMFAGate) — not one check per branch. Lockout is
	// set only by explicit user action (lock_mfa), never inferred here.
	if user.MFALockedAt != nil {
		log.Debug().Msg("User's MFA is locked, refusing login")
		return nil, nil, FailedPrecondition("your account's multi-factor authentication is locked; contact your administrator to regain access")
	}

	// Computed up-front (not just inside the TOTP/resolveMFAGate branch below)
	// so the email-OTP and SMS-OTP branches can offer it too: a registered
	// passkey satisfies MFA on its own (see webauthn.go's WebauthnLoginVerify),
	// so it must be offered as an alternative alongside whichever OTP method
	// gets challenged first — the same way it's already offered alongside
	// TOTP. Ignore a list error the same way the TOTP/resolveMFAGate branch
	// does: treat "couldn't check" as "found none" rather than failing login.
	webauthnCreds, _ := p.StorageProvider.ListWebauthnCredentialsByUserID(ctx, user.ID)
	hasWebauthnCredential := len(webauthnCreds) > 0

	// A verified Email-OTP second factor is challenged on enrollment alone,
	// independent of which identifier (email or phone) the caller logged in
	// with: a user who signed up with email, later verified a phone number,
	// and picked SMS/Email-OTP as their factor must still be challenged for it
	// on an email+password login. The code is sent to the account's own stored
	// contact (user.Email), not the login params, which may be empty for the
	// non-matching identifier.
	emailOTPAuthenticator, _ := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
	emailOTPEnrolled := emailOTPAuthenticator != nil && emailOTPAuthenticator.VerifiedAt != nil
	if effectiveMFAEnabled(p.Config, user) && isMFAEnabled && isMailOTPEnabled && isEmailServiceEnabled && emailOTPEnrolled {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := p.generateAndStoreOTP(ctx, user, expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to generate otp")
			return nil, nil, err
		}
		if err := p.setMFASession(meta, side, user.ID, expiresAt, params.Scope...); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, nil, err
		}
		asyncutil.Go(p.Log, func() {
			ctx := context.WithoutCancel(ctx)
			// exec it as go routine so that we can reduce the api latency
			if err := p.EmailProvider.SendEmail([]string{refs.StringValue(user.Email)}, constants.VerificationTypeOTP, map[string]any{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(p.Config),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug().Msg("Failed to send otp email")
			}
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		})
		res := &model.AuthResponse{
			Message:                  "Please check email inbox for the OTP",
			ShouldShowEmailOtpScreen: refs.NewBoolRef(true),
		}
		if hasWebauthnCredential {
			res.ShouldOfferWebauthnMfaVerify = refs.NewBoolRef(true)
		}
		return res, side, nil
	}
	// SMS-OTP twin of the email branch above: challenged on enrollment alone,
	// sent to user.PhoneNumber regardless of the login identifier. Email wins
	// deterministically if a user somehow enrolled both (email branch returns
	// first).
	smsOTPAuthenticator, _ := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeySMSOTPAuthenticator)
	smsOTPEnrolled := smsOTPAuthenticator != nil && smsOTPAuthenticator.VerifiedAt != nil
	if effectiveMFAEnabled(p.Config, user) && isMFAEnabled && isSMSOTPEnabled && isSMSServiceEnabled && smsOTPEnrolled {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := p.generateAndStoreOTP(ctx, user, expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to generate otp")
			return nil, nil, err
		}
		if err := p.setMFASession(meta, side, user.ID, expiresAt, params.Scope...); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, nil, err
		}
		asyncutil.Go(p.Log, func() {
			ctx := context.WithoutCancel(ctx)
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			if err := p.SMSProvider.SendSMS(refs.StringValue(user.PhoneNumber), smsBody.String()); err != nil {
				log.Debug().Msg("Failed to send sms")
			}
		})
		res := &model.AuthResponse{
			Message:                   "Please check text message for the OTP",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(true),
		}
		if hasWebauthnCredential {
			res.ShouldOfferWebauthnMfaVerify = refs.NewBoolRef(true)
		}
		return res, side, nil
	}
	// Gate runs whenever MFA applies at all -- NOT scoped to "TOTP
	// specifically is available" (that was the I1 bypass: a WebAuthn-only
	// enforced-MFA server, EnableTOTPLogin=false, skipped this block
	// entirely and issued tokens unconditionally). Mirrors webauthn.go's
	// WebauthnLoginVerify, which calls resolveMFAGate unconditionally and
	// only conditions the TOTP-specific parts of the response on
	// isTOTPLoginEnabled below.
	if isMFAEnabled {
		authenticator, authErr := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		totpVerified := authErr == nil && authenticator != nil && authenticator.VerifiedAt != nil
		// hasWebauthnCredential is computed once, up-front (see above) — reused
		// here rather than re-queried.
		authenticatorVerified := totpVerified || hasWebauthnCredential
		gate := resolveMFAGate(
			effectiveMFAEnabled(p.Config, user),
			p.Config.EnforceMFA,
			authenticatorVerified,
			user.HasSkippedMFASetupAt != nil,
		)
		switch gate {
		case mfaGateBlockVerify:
			expiresAt := time.Now().Add(3 * time.Minute).Unix()
			if err := p.setMFASession(meta, side, user.ID, expiresAt, params.Scope...); err != nil {
				log.Debug().Msg("Failed to set mfa session")
				return nil, nil, err
			}
			res := &model.AuthResponse{Message: `Proceed to mfa verification`}
			// Defense-in-depth: totpVerified can only be true if a TOTP row
			// exists, but a server could have disabled TOTP login after the
			// row was created (stale enrollment). Don't offer a screen the
			// user can no longer complete.
			if totpVerified && isTOTPLoginEnabled {
				res.ShouldShowTotpScreen = refs.NewBoolRef(true)
			}
			if hasWebauthnCredential {
				res.ShouldOfferWebauthnMfaVerify = refs.NewBoolRef(true)
			}
			return res, side, nil
		case mfaGateBlockEnroll:
			expiresAt := time.Now().Add(3 * time.Minute).Unix()
			if err := p.setMFASession(meta, side, user.ID, expiresAt, params.Scope...); err != nil {
				log.Debug().Msg("Failed to set mfa session")
				return nil, nil, err
			}
			res := &model.AuthResponse{
				Message:                     `Proceed to mfa setup`,
				ShouldOfferWebauthnMfaSetup: refs.NewBoolRef(p.Config.EnableWebauthnMFA),
				ShouldOfferEmailOtpMfaSetup: refs.NewBoolRef(p.Config.EnableEmailOTP && p.Config.IsEmailServiceEnabled),
				ShouldOfferSmsOtpMfaSetup:   refs.NewBoolRef(p.Config.EnableSMSOTP && p.Config.IsSMSServiceEnabled),
			}
			if isTOTPLoginEnabled {
				enrollment, err := p.generateTOTPEnrollment(ctx, user.ID)
				if err != nil {
					log.Debug().Msg("Failed to generate totp")
					return nil, nil, err
				}
				res.ShouldShowTotpScreen = refs.NewBoolRef(true)
				res.AuthenticatorScannerImage = refs.NewStringRef(enrollment.ScannerImage)
				res.AuthenticatorSecret = refs.NewStringRef(enrollment.Secret)
				res.AuthenticatorRecoveryCodes = enrollment.RecoveryCodes
			}
			return res, side, nil
		case mfaGateOfferAll:
			expiresAt := time.Now().Add(3 * time.Minute).Unix()
			if err := p.setMFASession(meta, side, user.ID, expiresAt, params.Scope...); err != nil {
				log.Debug().Msg("Failed to set mfa session")
				return nil, nil, err
			}
			res := &model.AuthResponse{
				Message:                     `Proceed to mfa setup`,
				ShouldOfferWebauthnMfaSetup: refs.NewBoolRef(p.Config.EnableWebauthnMFA),
				ShouldOfferEmailOtpMfaSetup: refs.NewBoolRef(p.Config.EnableEmailOTP && p.Config.IsEmailServiceEnabled),
				ShouldOfferSmsOtpMfaSetup:   refs.NewBoolRef(p.Config.EnableSMSOTP && p.Config.IsSMSServiceEnabled),
			}
			if isTOTPLoginEnabled {
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
		case mfaGateSkippedSetup:
			side.OfferMFASetupQuiet = true
		case mfaGateNone:
			// fall through, nothing to do
		}
	}

	code := ""
	codeChallenge := ""
	nonce := ""
	oidcNonce := ""
	authorizeRedirectURI := ""
	authorizeResource := ""
	authorizeClientID := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := p.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			// One owner for this positional format — see internal/codestate.
			if codestate.HasCode(authorizeState) {
				as := codestate.DecodeAuthorize(authorizeState)
				code = as.Code
				codeChallenge = as.Challenge
				oidcNonce = as.Nonce
				authorizeRedirectURI = as.RedirectURI
				authorizeResource = as.Resource
				authorizeClientID = as.ClientID
			} else {
				nonce = authorizeState
			}
			_ = p.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}

	if nonce == "" {
		nonce = uuid.New().String()
	}
	hostname := meta.HostURL
	// TokenProvider.CreateAuthToken takes *gin.Context but doesn't read from
	// it (only AccessToken-getter and ID-token-getter helpers in the same
	// file do). Synthesize a minimal gin.Context wrapping the inbound
	// *http.Request so the call works for both gin and non-gin transports.
	// TODO(grpc): refactor TokenProvider to take *http.Request directly.
	gcShim := &gin.Context{Request: meta.Request}
	authToken, err := p.TokenProvider.CreateAuthToken(gcShim, &token.AuthTokenConfig{
		User:        user,
		Roles:       roles,
		Scope:       scope,
		Nonce:       nonce,
		OIDCNonce:   oidcNonce,
		Code:        code,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		HostName:    hostname,
	})
	if err != nil {
		log.Debug().Msg("Failed to create auth token")
		return nil, nil, err
	}

	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := p.MemoryStoreProvider.SetState(code, codestate.EncodeCode(codestate.Code{
			Challenge:   codeChallenge,
			Session:     authToken.FingerPrintHash,
			Nonce:       oidcNonce,
			RedirectURI: authorizeRedirectURI,
			Resource:    authorizeResource,
			ClientID:    authorizeClientID,
		})); err != nil {
			log.Debug().Msg("Failed to set state")
			return nil, nil, err
		}
	}

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     `Logged in successfully`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
		User:        user.AsAPIUser(),
	}
	for _, c := range cookie.BuildSessionCookies(meta.HostURL, authToken.FingerPrintHash, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
		side.AddCookie(c)
	}
	sessionStoreKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	_ = p.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	_ = p.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		_ = p.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	asyncutil.Go(p.Log, func() {
		ctx := context.WithoutCancel(ctx)
		// Register event
		if isEmailLogin {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		} else {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}
		// Record session
		_ = p.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: meta.UserAgent,
			IP:        meta.IPAddress,
		})
	})
	metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusSuccess)
	metrics.ActiveSessions.Inc()
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditLoginSuccessEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeSession,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return res, side, nil
}
