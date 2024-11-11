package resolvers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	emailService "github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/models/db"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/smsproviders"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// SignupResolver is a resolver for signup mutation
func SignupResolver(ctx context.Context, params model.SignUpInput) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	isSignupDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableSignUp)
	if err != nil {
		log.Debug("Error getting signup disabled: ", err)
		isSignupDisabled = true
	}
	if isSignupDisabled {
		log.Debug("Signup is disabled")
		return res, fmt.Errorf(`signup is disabled for this instance`)
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}
	isMobileBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication)
	if err != nil {
		log.Debug("Error getting mobile basic auth disabled: ", err)
		isMobileBasicAuthDisabled = true
	}
	if params.ConfirmPassword != params.Password {
		log.Debug("Passwords do not match")
		return res, fmt.Errorf(`password and confirm password does not match`)
	}
	if err := validators.IsValidPassword(params.Password); err != nil {
		log.Debug("Invalid password")
		return res, err
	}
	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	if email == "" && phoneNumber == "" {
		log.Debug("Email or phone number is required")
		return res, fmt.Errorf(`email or phone number is required`)
	}
	isEmailSignup := email != ""
	isMobileSignup := phoneNumber != ""
	if isBasicAuthDisabled && isEmailSignup {
		log.Debug("Basic authentication is disabled")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isMobileBasicAuthDisabled && isMobileSignup {
		log.Debug("Mobile basic authentication is disabled")
		return res, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	if isEmailSignup && !validators.IsValidEmail(email) {
		log.Debug("Invalid email: ", params.Email)
		return res, fmt.Errorf(`invalid email address`)
	}
	if isMobileSignup && (phoneNumber == "" || len(phoneNumber) < 10) {
		log.Debug("Invalid phone number: ", phoneNumber)
		return res, fmt.Errorf(`invalid phone number`)
	}
	log := log.WithFields(log.Fields{
		"email":        email,
		"phone_number": phoneNumber,
	})
	// find user with email / phone number
	if isEmailSignup {
		existingUser, err := db.Provider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug("Failed to get user by email: ", err)
		}
		if existingUser != nil {
			if existingUser.EmailVerifiedAt != nil {
				// email is verified
				log.Debug("Email is already verified and signed up.")
				return res, fmt.Errorf(`%s has already signed up`, email)
			} else if existingUser.ID != "" && existingUser.EmailVerifiedAt == nil {
				log.Debug("Email is already signed up. Verification pending...")
				return res, fmt.Errorf("%s has already signed up. please complete the email verification process or reset the password", email)
			}
		}
	} else {
		existingUser, err := db.Provider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug("Failed to get user by phone number: ", err)
		}
		if existingUser != nil {
			if existingUser.PhoneNumberVerifiedAt != nil {
				// email is verified
				log.Debug("Phone number is already verified and signed up.")
				return res, fmt.Errorf(`%s has already signed up`, phoneNumber)
			} else if existingUser.ID != "" && existingUser.PhoneNumberVerifiedAt == nil {
				log.Debug("Phone number is already signed up. Verification pending...")
				return res, fmt.Errorf("%s has already signed up. please complete the phone number verification process or reset the password", phoneNumber)
			}
		}
	}

	inputRoles := []string{}
	if len(params.Roles) > 0 {
		// check if roles exists
		rolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyRoles)
		roles := []string{}
		if err != nil {
			log.Debug("Error getting roles: ", err)
			return res, err
		} else {
			roles = strings.Split(rolesString, ",")
		}
		if !validators.IsValidRoles(params.Roles, roles) {
			log.Debug("Invalid roles: ", params.Roles)
			return res, fmt.Errorf(`invalid roles`)
		} else {
			inputRoles = params.Roles
		}
	} else {
		inputRolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
		if err != nil {
			log.Debug("Error getting default roles: ", err)
			return res, err
		} else {
			inputRoles = strings.Split(inputRolesString, ",")
		}
	}
	user := &models.User{}
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

	isMFAEnforced, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyEnforceMultiFactorAuthentication)
	if err != nil {
		log.Debug("MFA service not enabled: ", err)
		isMFAEnforced = false
	}

	if isMFAEnforced {
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	}

	if params.AppData != nil {
		appDataString := ""
		appDataBytes, err := json.Marshal(params.AppData)
		if err != nil {
			log.Debug("failed to marshall source app_data: ", err)
			return nil, errors.New("malformed app_data")
		}
		appDataString = string(appDataBytes)
		user.AppData = &appDataString
	}
	isEmailVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification)
	if err != nil {
		log.Debug("Error getting email verification disabled: ", err)
		isEmailVerificationDisabled = true
	}
	if isEmailVerificationDisabled && isEmailSignup {
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}
	disablePhoneVerification, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
	if disablePhoneVerification && isMobileSignup {
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}
	isSMSServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsSMSServiceEnabled)
	if err != nil || !isSMSServiceEnabled {
		log.Debug("SMS service not enabled: ", err)
	}
	user, err = db.Provider.AddUser(ctx, user)
	if err != nil {
		log.Debug("Failed to add user: ", err)
		return res, err
	}
	roles := strings.Split(user.Roles, ",")
	userToReturn := user.AsAPIUser()
	hostname := parsers.GetHost(gc)
	if !isEmailVerificationDisabled && isEmailSignup {
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug("Failed to generate nonce: ", err)
			return res, err
		}
		verificationType := constants.VerificationTypeBasicAuthSignup
		redirectURL := parsers.GetAppURL(gc)
		if params.RedirectURI != nil {
			redirectURL = *params.RedirectURI
		}
		verificationToken, err := token.CreateVerificationToken(email, verificationType, hostname, nonceHash, redirectURL)
		if err != nil {
			log.Debug("Failed to create verification token: ", err)
			return res, err
		}
		_, err = db.Provider.AddVerificationRequest(ctx, &models.VerificationRequest{
			Token:       verificationToken,
			Identifier:  verificationType,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       email,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		})
		if err != nil {
			log.Debug("Failed to add verification request: ", err)
			return res, err
		}
		// exec it as go routine so that we can reduce the api latency
		go func() {
			// exec it as go routine so that we can reduce the api latency
			emailService.SendEmail([]string{email}, constants.VerificationTypeBasicAuthSignup, map[string]interface{}{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(),
				"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
			})
			utils.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()

		return &model.AuthResponse{
			Message: `Verification email has been sent. Please check your inbox`,
			User:    userToReturn,
		}, nil
	} else if !disablePhoneVerification && isSMSServiceEnabled && isMobileSignup {
		duration, _ := time.ParseDuration("10m")
		smsCode := utils.GenerateOTP()
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(smsCode)
		expiresAt := time.Now().Add(duration).Unix()
		_, err = db.Provider.UpsertOTP(ctx, &models.OTP{
			PhoneNumber: phoneNumber,
			Otp:         smsCode,
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug("error while upserting OTP: ", err.Error())
			return nil, err
		}
		mfaSession := uuid.NewString()
		err = memorystore.Provider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug("Failed to add mfasession: ", err)
			return nil, err
		}
		cookie.SetMfaSession(gc, mfaSession)
		go func() {
			smsproviders.SendSMS(phoneNumber, smsBody.String())
			utils.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
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
		authorizeState, _ := memorystore.Provider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
			} else {
				nonce = authorizeState
			}
			go memorystore.Provider.RemoveState(refs.StringValue(params.State))
		}
	}

	if nonce == "" {
		nonce = uuid.New().String()
	}

	authToken, err := token.CreateAuthToken(gc, user, roles, scope, constants.AuthRecipeMethodBasicAuth, nonce, code)
	if err != nil {
		log.Debug("Failed to create auth token: ", err)
		return res, err
	}

	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
			log.Debug("SetState failed: ", err)
			return res, err
		}
	}

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `Signed up successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		ExpiresIn:   &expiresIn,
		User:        userToReturn,
	}

	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	cookie.SetSession(gc, authToken.FingerPrintHash)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		utils.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		if isEmailSignup {
			utils.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		} else {
			utils.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}

		db.Provider.AddSession(ctx, &models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	return res, nil
}
