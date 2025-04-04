package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// SignUp is the method to singup user
// Permission: none
func (g *graphqlProvider) SignUp(ctx context.Context, params *model.SignUpInput) (*model.AuthResponse, error) {
	log := g.Log.With().Str("func", "SignUp").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, fmt.Errorf(`email or phone number is required`)
	}

	isSignupDisabled := g.Config.DisableSignup
	if isSignupDisabled {
		log.Debug().Msg("Signup is disabled")
		return nil, fmt.Errorf(`signup is disabled for this instance`)
	}

	isBasicAuthDisabled := g.Config.DisableBasicAuthentication
	isMobileBasicAuthDisabled := g.Config.DisableMobileBasicAuthentication
	if params.ConfirmPassword != params.Password {
		log.Debug().Msg("Passwords do not match")
		return nil, fmt.Errorf(`password and confirm password does not match`)
	}
	if err := validators.IsValidPassword(params.Password, g.Config.DisableStrongPassword); err != nil {
		log.Debug().Msg("Invalid password")
		return nil, err
	}

	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	isEmailSignup := email != ""
	isMobileSignup := phoneNumber != ""
	if isBasicAuthDisabled && isEmailSignup {
		log.Debug().Msg("Basic authentication is disabled")
		return nil, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isMobileBasicAuthDisabled && isMobileSignup {
		log.Debug().Msg("Mobile basic authentication is disabled")
		return nil, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	if isEmailSignup && !validators.IsValidEmail(email) {
		log.Debug().Msg("Invalid email")
		return nil, fmt.Errorf(`invalid email address`)
	}
	if isMobileSignup && (phoneNumber == "" || len(phoneNumber) < 10) {
		log.Debug().Msg("Invalid phone number")
		return nil, fmt.Errorf(`invalid phone number`)
	}
	// find user with email / phone number
	if isEmailSignup {
		existingUser, err := g.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
		}
		if existingUser != nil {
			if existingUser.EmailVerifiedAt != nil {
				// email is verified
				log.Debug().Msg("Email is already verified and signed up.")
				return nil, fmt.Errorf(`%s has already signed up`, email)
			} else if existingUser.ID != "" && existingUser.EmailVerifiedAt == nil {
				log.Debug().Msg("Email is already signed up. Verification pending...")
				return nil, fmt.Errorf("%s has already signed up. please complete the email verification process or reset the password", email)
			}
		}
	} else {
		existingUser, err := g.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
		}
		if existingUser != nil {
			if existingUser.PhoneNumberVerifiedAt != nil {
				// email is verified
				log.Debug().Msg("Phone number is already verified and signed up.")
				return nil, fmt.Errorf(`%s has already signed up`, phoneNumber)
			} else if existingUser.ID != "" && existingUser.PhoneNumberVerifiedAt == nil {
				log.Debug().Msg("Phone number is already signed up. Verification pending...")
				return nil, fmt.Errorf("%s has already signed up. please complete the phone number verification process or reset the password", phoneNumber)
			}
		}
	}

	inputRoles := params.Roles
	if len(inputRoles) > 0 {
		// check if roles exists
		roles := g.Config.Roles
		if !validators.IsValidRoles(inputRoles, roles) {
			log.Debug().Err(err).Strs("roles", params.Roles).Msg("Invalid roles")
			return nil, fmt.Errorf(`invalid roles`)
		}
	} else {
		inputRoles = g.Config.DefaultRoles
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

	isMFAEnforced := g.Config.EnforceMFA
	if isMFAEnforced {
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	}

	if params.AppData != nil {
		appDataString := ""
		appDataBytes, err := json.Marshal(params.AppData)
		if err != nil {
			log.Debug().Msg("failed to marshall source app_data")
			return nil, errors.New("malformed app_data")
		}
		appDataString = string(appDataBytes)
		user.AppData = &appDataString
	}
	isEmailServiceEnabled := g.Config.IsEmailServiceEnabled
	isEmailVerificationDisabled := g.Config.DisableEmailVerification || !isEmailServiceEnabled
	if (isEmailVerificationDisabled) && isEmailSignup {
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}
	isSMSServiceEnabled := g.Config.IsSMSServiceEnabled
	isPhoneVerificationDisabled := g.Config.DisablePhoneVerification || !isSMSServiceEnabled
	if (isPhoneVerificationDisabled) && isMobileSignup {
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}
	user, err = g.StorageProvider.AddUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("failed to add user")
		return nil, err
	}
	roles := strings.Split(user.Roles, ",")
	userToReturn := user.AsAPIUser()
	hostname := parsers.GetHost(gc)
	if !isEmailVerificationDisabled && isEmailSignup {
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate nonce")
			return nil, err
		}
		verificationType := constants.VerificationTypeBasicAuthSignup
		redirectURL := parsers.GetAppURL(gc)
		if params.RedirectURI != nil {
			redirectURL = *params.RedirectURI
		}
		verificationToken, err := g.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			Nonce:       nonceHash,
			HostName:    hostname,
			User:        user,
			LoginMethod: constants.AuthRecipeMethodBasicAuth,
		}, redirectURL, verificationType)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create verification token")
			return nil, err
		}
		_, err = g.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
			Token:       verificationToken,
			Identifier:  verificationType,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       email,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add verification request")
			return nil, err
		}
		// exec it as go routine so that we can reduce the api latency
		go func() {
			// exec it as go routine so that we can reduce the api latency
			g.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeBasicAuthSignup, map[string]interface{}{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(g.Config),
				"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
			})
			g.EventsProvider.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()

		return &model.AuthResponse{
			Message: `Verification email has been sent. Please check your inbox`,
		}, nil
	} else if !isPhoneVerificationDisabled && isMobileSignup {
		duration, _ := time.ParseDuration("10m")
		smsCode := utils.GenerateOTP()
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(smsCode)
		expiresAt := time.Now().Add(duration).Unix()
		_, err = g.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			PhoneNumber: phoneNumber,
			Otp:         smsCode,
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug().Err(err).Msg("error while upserting OTP")
			return nil, err
		}
		mfaSession := uuid.NewString()
		err = g.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add mfasession")
			return nil, err
		}
		cookie.SetMfaSession(gc, mfaSession)
		go func() {
			g.SMSProvider.SendSMS(phoneNumber, smsBody.String())
			g.EventsProvider.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}()
		return &model.AuthResponse{
			Message:                   "Please check the OTP in your inbox",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(true),
		}, nil
	}
	scope := []string{"openid", "email", "profile"}
	if params.Scope != nil && len(scope) > 0 {
		scope = params.Scope
	}

	code := ""
	codeChallenge := ""
	nonce := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := g.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
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
		Nonce:       nonce,
		Code:        code,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		HostName:    hostname,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create auth token")
		return nil, err
	}

	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := g.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
			log.Debug().Err(err).Msg("SetState failed")
			return nil, err
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
	cookie.SetSession(gc, authToken.FingerPrintHash)
	g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		g.EventsProvider.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		if isEmailSignup {
			g.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		} else {
			g.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}

		if err := g.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		}); err != nil {
			log.Debug().Err(err).Msg("Failed to add session")
		}
	}()

	return res, nil
}
