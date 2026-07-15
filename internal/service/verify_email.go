package service

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// VerifyEmail verifies a user's email using a verification token, completing
// signup (or a magic-link login) and issuing a session. Transport-agnostic
// port of graphqlProvider.VerifyEmail.
//
// Permissions: none.
func (p *provider) VerifyEmail(ctx context.Context, meta RequestMetadata, params *model.VerifyEmailRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "VerifyEmail").Logger()
	side := &ResponseSideEffects{}

	verificationRequest, err := p.StorageProvider.GetVerificationRequestByToken(ctx, params.Token)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetVerificationRequestByToken")
		return nil, nil, InvalidArgument(`invalid verification token`)
	}

	// verify if token exists in db
	hostname := meta.HostURL
	claim, err := p.TokenProvider.ParseJWTToken(params.Token)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse jwt token")
		return nil, nil, InvalidArgument(`invalid verification token`)
	}

	if ok, err := p.TokenProvider.ValidateJWTClaims(claim, &token.AuthTokenConfig{
		HostName: hostname,
		Nonce:    verificationRequest.Nonce,
		User: &schemas.User{
			Email: &verificationRequest.Email,
		},
	}); !ok || err != nil {
		log.Debug().Err(err).Msg("Failed to validate jwt claims")
		return nil, nil, InvalidArgument(`invalid verification token`)
	}

	email := claim["sub"].(string)
	log.Debug().Str("email", email).Msg("Email verified successfully")
	user, err := p.StorageProvider.GetUserByEmail(ctx, email)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetUserByEmail")
		return nil, nil, err
	}

	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, nil, FailedPrecondition("user access has been revoked")
	}

	isMFAEnabled := p.Config.EnableMFA
	isTOTPLoginEnabled := p.Config.EnableTOTPLogin

	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		// Reached only via a signed verification token mailed to the user's
		// inbox — a possession proof of this exact account, so Verified.
		err = p.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeVerified, expiresAt)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to set mfa session")
			return err
		}
		for _, c := range cookie.BuildMfaSessionCookies(hostname, mfaSession, p.Config.AppCookieSecure) {
			side.AddCookie(c)
		}
		return nil
	}

	// If mfa enabled and also totp enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && isMFAEnabled && isTOTPLoginEnabled {
		expiresAt := time.Now().Add(3 * time.Minute).Unix()
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug().Err(err).Msg("Failed to set mfa session")
			return nil, nil, err
		}
		authenticator, err := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		if err != nil || authenticator == nil || authenticator.VerifiedAt == nil {
			// generate totp
			// Generate a base64 URL and initiate the registration for TOTP
			authConfig, err := p.AuthenticatorProvider.Generate(ctx, user.ID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to generate totp")
				return nil, nil, err
			}
			recoveryCodes := []*string{}
			for _, code := range authConfig.RecoveryCodes {
				recoveryCodes = append(recoveryCodes, refs.NewStringRef(code))
			}
			// when user is first time registering for totp
			res := &model.AuthResponse{
				Message:                    `Proceed to totp verification screen`,
				ShouldShowTotpScreen:       refs.NewBoolRef(true),
				AuthenticatorScannerImage:  refs.NewStringRef(authConfig.ScannerImage),
				AuthenticatorSecret:        refs.NewStringRef(authConfig.Secret),
				AuthenticatorRecoveryCodes: recoveryCodes,
			}
			return res, side, nil
		} else {
			// when user is already register for totp
			res := &model.AuthResponse{
				Message:              `Proceed to totp screen`,
				ShouldShowTotpScreen: refs.NewBoolRef(true),
			}
			return res, side, nil
		}
	}

	isSignUp := false
	if user.EmailVerifiedAt == nil {
		isSignUp = true
		// update email_verified_at in users table
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
		user, err = p.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("failed UpdateUser")
			return nil, nil, err
		}
	}
	// delete from verification table
	err = p.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest)
	if err != nil {
		log.Debug().Err(err).Msg("failed DeleteVerificationRequest")
		return nil, nil, err
	}

	loginMethod := constants.AuthRecipeMethodBasicAuth
	if verificationRequest.Identifier == constants.VerificationTypeMagicLinkLogin {
		loginMethod = constants.AuthRecipeMethodMagicLinkLogin
	}

	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	code := ""
	// Not required as /oauth/token cannot be resumed from other tab
	// codeChallenge := ""
	nonce := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := p.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				// Not required as /oauth/token cannot be resumed from other tab
				// codeChallenge = authorizeStateSplit[1]
			} else {
				nonce = authorizeState
			}
			go func() { _ = p.MemoryStoreProvider.RemoveState(refs.StringValue(params.State)) }()
		}
	}
	if nonce == "" {
		nonce = uuid.New().String()
	}
	// TokenProvider.CreateAuthToken takes *gin.Context but doesn't read from it.
	// Synthesize a minimal gin.Context wrapping the inbound *http.Request so the
	// call works for both gin and non-gin transports.
	gcShim := &gin.Context{Request: meta.Request}
	authToken, err := p.TokenProvider.CreateAuthToken(gcShim, &token.AuthTokenConfig{
		HostName:    hostname,
		User:        user,
		Roles:       roles,
		Scope:       scope,
		LoginMethod: loginMethod,
		Nonce:       nonce,
		Code:        code,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create auth token")
		return nil, nil, err
	}

	// Code challenge could be optional if PKCE flow is not used
	// Not required as /oauth/token cannot be resumed from other tab
	// if code != "" {
	// 	if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
	// 		log.Debug("SetState failed: ", err)
	// 		return nil, err
	// 	}
	// }
	go func() {
		ctx := context.WithoutCancel(ctx)
		if isSignUp {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, loginMethod, user)
			// User is also logged in with signup
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		} else {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		}

		if err := p.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: meta.UserAgent,
			IP:        meta.IPAddress,
		}); err != nil {
			log.Debug().Err(err).Msg("Failed to add session")
		}
	}()
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
	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     `Email verified successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
		User:        user.AsAPIUser(),
	}

	sessionKey := loginMethod + ":" + user.ID
	for _, c := range cookie.BuildSessionCookies(hostname, authToken.FingerPrintHash, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
		side.AddCookie(c)
	}
	_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}
	return res, side, nil
}
