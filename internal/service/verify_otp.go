package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// VerifyOtp is the method to verify OTP
// authorized otp request
func (s *service) VerifyOTP(ctx context.Context, params *model.VerifyOTPRequest) (*model.AuthResponse, error) {
	log := s.Log.With().Str("func", "VerifyOTP").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	mfaSession, err := cookie.GetMfaSession(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, fmt.Errorf(`invalid session: %s`, err.Error())
	}

	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, fmt.Errorf(`email or phone number is required`)
	}
	isEmailVerification := email != ""
	isMobileVerification := phoneNumber != ""
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	// Get user by email or phone number
	var user *schemas.User
	if isEmailVerification {
		user, err = s.StorageProvider.GetUserByEmail(ctx, refs.StringValue(params.Email))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
		}
	} else {
		user, err = s.StorageProvider.GetUserByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
		}
	}
	if user == nil || err != nil {
		log.Debug().Err(err).Msg("User not found")
		return nil, fmt.Errorf(`user not found`)
	}
	// Verify OTP based on TOPT or OTP
	if refs.BoolValue(params.IsTotp) {
		status, err := s.AuthenticatorProvider.Validate(ctx, params.Otp, user.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to validate passcode")
			return nil, fmt.Errorf("error while validating passcode")
		}
		if !status {
			log.Debug().Msg("Failed to verify otp request: Incorrect value")
			log.Info().Msg("Checking if otp is recovery code")
			// Check if otp is recovery code
			isValidRecoveryCode, err := s.AuthenticatorProvider.ValidateRecoveryCode(ctx, params.Otp, user.ID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to validate recovery code")
				return nil, fmt.Errorf("error while validating recovery code")
			}
			if !isValidRecoveryCode {
				log.Debug().Msg("Failed to verify otp request: Incorrect value")
				return nil, fmt.Errorf(`invalid otp`)
			}
		}
	} else {
		var otp *schemas.OTP
		if isEmailVerification {
			otp, err = s.StorageProvider.GetOTPByEmail(ctx, refs.StringValue(params.Email))
			if err != nil {
				log.Debug().Err(err).Msg("Failed to get otp request for email")
			}
		} else {
			otp, err = s.StorageProvider.GetOTPByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
			if err != nil {
				log.Debug().Err(err).Msg("Failed to get otp request for phone number")
			}
		}
		if otp == nil && err != nil {
			log.Debug().Msg("OTP not found")
			return nil, fmt.Errorf(`OTP not found`)
		}
		if params.Otp != otp.Otp {
			log.Debug().Msg("Failed to verify otp request: OTP mismatch")
			return nil, fmt.Errorf(`invalid otp`)
		}
		expiresIn := otp.ExpiresAt - time.Now().Unix()
		if expiresIn < 0 {
			log.Debug().Msg("OTP expired")
			return nil, fmt.Errorf("otp expired")
		}
		if err := s.StorageProvider.DeleteOTP(gc, otp); err != nil {
			log.Debug().Err(err).Msg("Failed to delete otp")
		}
	}

	if _, err := s.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession); err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, fmt.Errorf(`invalid session: %s`, err.Error())
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
		user, err = s.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to update user")
			return nil, err
		}
	}
	loginMethod := constants.AuthRecipeMethodBasicAuth
	if isMobileVerification {
		loginMethod = constants.AuthRecipeMethodMobileOTP
	}
	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	code := ""
	codeChallenge := ""
	nonce := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := s.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
			} else {
				nonce = authorizeState
			}
			go s.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}
	if nonce == "" {
		nonce = uuid.New().String()
	}
	// user, roles, scope, loginMethod, nonce, code
	authToken, err := s.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
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
	if code != "" {
		if err := s.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
			log.Debug().Err(err).Msg("Failed to set state")
			return nil, err
		}
	}

	go func() {
		if isSignUp {
			s.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, loginMethod, user)
			// User is also logged in with signup
			s.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		} else {
			s.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		}

		if err := s.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		}); err != nil {
			log.Debug().Err(err).Msg("Failed to add session")
		}
	}()

	authTokenExpiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if authTokenExpiresIn <= 0 {
		authTokenExpiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     `OTP verified successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &authTokenExpiresIn,
		User:        user.AsAPIUser(),
	}

	sessionKey := loginMethod + ":" + user.ID
	cookie.SetSession(gc, authToken.FingerPrintHash)
	s.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	s.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		s.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}
	return res, nil
}
