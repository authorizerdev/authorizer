package graphql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// VerifyEmail is the method to verify email
// Permission: none
func (g *graphqlProvider) VerifyEmail(ctx context.Context, params *model.VerifyEmailInput) (*model.AuthResponse, error) {
	log := g.Log.With().Str("func", "VerifyEmail").Logger()

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	verificationRequest, err := g.StorageProvider.GetVerificationRequestByToken(ctx, params.Token)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetVerificationRequestByToken")
		return nil, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	// verify if token exists in db
	hostname := parsers.GetHost(gc)
	claim, err := g.TokenProvider.ParseJWTToken(params.Token)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse jwt token")
		return nil, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	if ok, err := g.TokenProvider.ValidateJWTClaims(claim, &token.AuthTokenConfig{
		HostName: hostname,
		Nonce:    verificationRequest.Nonce,
		User: &schemas.User{
			Email: &verificationRequest.Email,
		},
	}); !ok || err != nil {
		log.Debug().Err(err).Msg("Failed to validate jwt claims")
		return nil, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	email := claim["sub"].(string)
	log.Debug().Str("email", email).Msg("Email verified successfully")
	user, err := g.StorageProvider.GetUserByEmail(ctx, email)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetUserByEmail")
		return nil, err
	}

	isMFADisabled := g.Config.DisableMFA
	isTOTPLoginDisabled := g.Config.DisableTOTPLogin

	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		err = g.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to set mfa session")
			return err
		}
		cookie.SetMfaSession(gc, mfaSession)
		return nil
	}

	// If mfa enabled and also totp enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && !isMFADisabled && !isTOTPLoginDisabled {
		expiresAt := time.Now().Add(3 * time.Minute).Unix()
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug().Err(err).Msg("Failed to set mfa session")
			return nil, err
		}
		authenticator, err := g.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		if err != nil || authenticator == nil || authenticator.VerifiedAt == nil {
			// generate totp
			// Generate a base64 URL and initiate the registration for TOTP
			authConfig, err := g.AuthenticatorProvider.Generate(ctx, user.ID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to generate totp")
				return nil, err
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
			return res, nil
		} else {
			//when user is already register for totp
			res := &model.AuthResponse{
				Message:              `Proceed to totp screen`,
				ShouldShowTotpScreen: refs.NewBoolRef(true),
			}
			return res, nil
		}
	}

	isSignUp := false
	if user.EmailVerifiedAt == nil {
		isSignUp = true
		// update email_verified_at in users table
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
		user, err = g.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("failed UpdateUser")
			return nil, err
		}
	}
	// delete from verification table
	err = g.StorageProvider.DeleteVerificationRequest(gc, verificationRequest)
	if err != nil {
		log.Debug().Err(err).Msg("failed DeleteVerificationRequest")
		return nil, err
	}

	loginMethod := constants.AuthRecipeMethodBasicAuth
	if loginMethod == constants.VerificationTypeMagicLinkLogin {
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
		authorizeState, _ := g.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				// Not required as /oauth/token cannot be resumed from other tab
				// codeChallenge = authorizeStateSplit[1]
			} else {
				nonce = authorizeState
			}
			go g.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}
	if nonce == "" {
		nonce = uuid.New().String()
	}
	authToken, err := g.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
		User:        user,
		Roles:       roles,
		Scope:       scope,
		LoginMethod: loginMethod,
		Nonce:       nonce,
		Code:        code,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create auth token")
		return nil, err
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
		if isSignUp {
			g.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, loginMethod, user)
			// User is also logged in with signup
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		} else {
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		}

		if err := g.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		}); err != nil {
			log.Debug().Err(err).Msg("Failed to add session")
		}
	}()
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
	cookie.SetSession(gc, authToken.FingerPrintHash)
	g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}
	return res, nil
}
