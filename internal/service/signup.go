package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// dummyHash is a precomputed bcrypt hash used to equalise the response time
// of the "user exists" path with the "new signup" path, preventing account
// enumeration via timing.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

// SignUp registers a new user. Public — no authentication required.
//
// Transport-agnostic: takes a RequestMetadata (host, IP, UA) instead of
// reaching into gin.Context, and returns cookie side-effects for the
// transport to apply.
func (p *provider) SignUp(ctx context.Context, meta RequestMetadata, params *model.SignUpRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "SignUp").Logger()
	side := &ResponseSideEffects{}

	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, nil, InvalidArgument("email or phone number is required")
	}

	isSignupEnabled := p.Config.EnableSignup
	if !isSignupEnabled {
		log.Debug().Msg("Signup is disabled")
		return nil, nil, FailedPrecondition("signup is disabled for this instance")
	}

	isBasicAuthEnabled := p.Config.EnableBasicAuthentication
	isMobileBasicAuthEnabled := p.Config.EnableMobileBasicAuthentication
	if params.ConfirmPassword != params.Password {
		log.Debug().Msg("Passwords do not match")
		return nil, nil, InvalidArgument("password and confirm password does not match")
	}
	if err := validators.IsValidPassword(params.Password, !p.Config.EnableStrongPassword); err != nil {
		log.Debug().Msg("Invalid password")
		return nil, nil, InvalidArgument(err.Error())
	}

	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	isEmailSignup := email != ""
	isMobileSignup := phoneNumber != ""
	if !isBasicAuthEnabled && isEmailSignup {
		log.Debug().Msg("Basic authentication is disabled")
		return nil, nil, FailedPrecondition("basic authentication is disabled for this instance")
	}
	if !isMobileBasicAuthEnabled && isMobileSignup {
		log.Debug().Msg("Mobile basic authentication is disabled")
		return nil, nil, FailedPrecondition("mobile basic authentication is disabled for this instance")
	}
	if isEmailSignup && !validators.IsValidEmail(email) {
		log.Debug().Msg("Invalid email")
		return nil, nil, InvalidArgument("invalid email address")
	}
	if isMobileSignup && (phoneNumber == "" || len(phoneNumber) < 10) {
		log.Debug().Msg("Invalid phone number")
		return nil, nil, InvalidArgument("invalid phone number")
	}
	// find user with email / phone number
	if isEmailSignup {
		existingUser, err := p.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
		}
		if existingUser != nil && (existingUser.EmailVerifiedAt != nil || existingUser.ID != "") {
			log.Debug().Msg("Email is already signed up.")
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte("timing-equalization"))
			return nil, nil, InvalidArgument("signup failed. please check your credentials or try a different method")
		}
	} else {
		existingUser, err := p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
		}
		if existingUser != nil && (existingUser.PhoneNumberVerifiedAt != nil || existingUser.ID != "") {
			log.Debug().Msg("Phone number is already signed up.")
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte("timing-equalization"))
			return nil, nil, InvalidArgument("signup failed. please check your credentials or try a different method")
		}
	}

	inputRoles := params.Roles
	if len(inputRoles) > 0 {
		// check if roles exists
		roles := p.Config.Roles
		if !validators.IsValidRoles(inputRoles, roles) {
			log.Debug().Strs("roles", params.Roles).Msg("Invalid roles")
			return nil, nil, InvalidArgument("invalid roles")
		}
	} else {
		inputRoles = p.Config.DefaultRoles
	}
	user := &schemas.User{}
	user.Roles = strings.Join(inputRoles, ",")
	password, _ := crypto.EncryptPassword(params.Password)
	user.Password = &password
	if email != "" {
		user.SignupMethods = constants.AuthRecipeMethodBasicAuth
		user.Email = &email
	}
	if params.GivenName != nil {
		user.GivenName = params.GivenName
	}

	if params.FamilyName != nil {
		user.FamilyName = params.FamilyName
	}

	if params.MiddleName != nil {
		user.MiddleName = params.MiddleName
	}

	if params.Nickname != nil {
		user.Nickname = params.Nickname
	}

	if params.Gender != nil {
		user.Gender = params.Gender
	}

	if params.Birthdate != nil {
		user.Birthdate = params.Birthdate
	}

	if phoneNumber != "" {
		user.SignupMethods = constants.AuthRecipeMethodMobileBasicAuth
		user.PhoneNumber = refs.NewStringRef(phoneNumber)
	}

	if params.Picture != nil {
		user.Picture = params.Picture
	}

	if params.IsMultiFactorAuthEnabled != nil {
		user.IsMultiFactorAuthEnabled = params.IsMultiFactorAuthEnabled
	}

	isMFAEnforced := p.Config.EnforceMFA
	if isMFAEnforced {
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	}

	if params.AppData != nil {
		appDataString := ""
		appDataBytes, err := json.Marshal(params.AppData)
		if err != nil {
			log.Debug().Msg("failed to marshall source app_data")
			return nil, nil, InvalidArgument("malformed app_data")
		}
		appDataString = string(appDataBytes)
		user.AppData = &appDataString
	}
	isEmailServiceEnabled := p.Config.IsEmailServiceEnabled
	isEmailVerificationEnabled := p.Config.EnableEmailVerification && isEmailServiceEnabled
	if !isEmailVerificationEnabled && isEmailSignup {
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}
	isSMSServiceEnabled := p.Config.IsSMSServiceEnabled
	isPhoneVerificationEnabled := p.Config.EnablePhoneVerification && isSMSServiceEnabled
	if !isPhoneVerificationEnabled && isMobileSignup {
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}
	user, err := p.StorageProvider.AddUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("failed to add user")
		return nil, nil, err
	}
	roles := strings.Split(user.Roles, ",")
	userToReturn := user.AsAPIUser()
	hostname := meta.HostURL
	if isEmailVerificationEnabled && isEmailSignup {
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate nonce")
			return nil, nil, err
		}
		verificationType := constants.VerificationTypeBasicAuthSignup
		redirectURL := hostname + "/app"
		if params.RedirectURI != nil {
			redirectURL = *params.RedirectURI
			if !validators.IsValidRedirectURI(redirectURL, p.Config.AllowedOrigins, hostname) {
				log.Debug().Msg("Invalid redirect URI")
				return nil, nil, InvalidArgument("invalid redirect URI")
			}
		}
		verificationToken, err := p.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			Nonce:       nonceHash,
			HostName:    hostname,
			User:        user,
			LoginMethod: constants.AuthRecipeMethodBasicAuth,
		}, redirectURL, verificationType)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create verification token")
			return nil, nil, err
		}
		_, err = p.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
			Token:       verificationToken,
			Identifier:  verificationType,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       email,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add verification request")
			return nil, nil, err
		}
		// exec it as go routine so that we can reduce the api latency
		go func() {
			_ = p.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeBasicAuthSignup, map[string]interface{}{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(p.Config),
				"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
			})
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()

		return &model.AuthResponse{
			Message: `Verification email has been sent. Please check your inbox`,
		}, side, nil
	} else if isPhoneVerificationEnabled && isMobileSignup {
		duration, _ := time.ParseDuration("10m")
		smsCode, err := utils.GenerateOTP()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate OTP")
			return nil, nil, err
		}
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(smsCode)
		expiresAt := time.Now().Add(duration).Unix()
		// Store the HMAC digest of the OTP; smsCode (plaintext) is sent
		// over SMS by the existing smsBody above.
		_, err = p.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			PhoneNumber: phoneNumber,
			Otp:         crypto.HashOTP(smsCode, p.Config.JWTSecret),
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug().Err(err).Msg("error while upserting OTP")
			return nil, nil, err
		}
		mfaSession := uuid.NewString()
		err = p.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add mfasession")
			return nil, nil, err
		}
		for _, c := range cookie.BuildMfaSessionCookies(hostname, mfaSession, p.Config.AppCookieSecure) {
			side.AddCookie(c)
		}
		go func() {
			_ = p.SMSProvider.SendSMS(phoneNumber, smsBody.String())
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}()
		return &model.AuthResponse{
			Message:                   "Please check the OTP in your inbox",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(true),
		}, side, nil
	}
	scope := []string{"openid", "email", "profile"}
	if len(params.Scope) > 0 {
		scope = params.Scope
	}

	code := ""
	codeChallenge := ""
	nonce := ""
	oidcNonce := ""
	authorizeRedirectURI := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := p.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
				if len(authorizeStateSplit) > 2 {
					oidcNonce = authorizeStateSplit[2]
				}
				if len(authorizeStateSplit) > 3 {
					authorizeRedirectURI = authorizeStateSplit[3]
				}
			} else {
				nonce = authorizeState
			}
			_ = p.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}

	if nonce == "" {
		nonce = uuid.New().String()
	}
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
		log.Debug().Err(err).Msg("Failed to create auth token")
		return nil, nil, err
	}

	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := p.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash+"@@"+oidcNonce+"@@"+authorizeRedirectURI); err != nil {
			log.Debug().Err(err).Msg("SetState failed")
			return nil, nil, err
		}
	}

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     `Signed up successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		ExpiresIn:   &expiresIn,
		User:        userToReturn,
	}

	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	for _, c := range cookie.BuildSessionCookies(hostname, authToken.FingerPrintHash, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
		side.AddCookie(c)
	}
	_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	ipAddress := meta.IPAddress
	userAgent := meta.UserAgent
	go func() {
		_ = p.EventsProvider.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		if isEmailSignup {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		} else {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}

		if err := p.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: userAgent,
			IP:        ipAddress,
		}); err != nil {
			log.Debug().Err(err).Msg("Failed to add session")
		}
	}()
	metrics.RecordAuthEvent(metrics.EventSignup, metrics.StatusSuccess)
	metrics.ActiveSessions.Inc()
	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditSignupEvent,
		ActorID:      user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})

	return res, side, nil
}
