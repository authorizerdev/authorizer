package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/smsproviders"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
)

// MobileSignupResolver is a resolver for mobile_basic_auth_signup mutation
func MobileSignupResolver(ctx context.Context, params *model.MobileSignUpInput) (*model.AuthResponse, error) {
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

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}

	if isBasicAuthDisabled {
		log.Debug("Mobile based Basic authentication is disabled")
		return res, fmt.Errorf(`phone number based basic authentication is disabled for this instance`)
	}

	if params.ConfirmPassword != params.Password {
		log.Debug("Passwords do not match")
		return res, fmt.Errorf(`password and confirm password does not match`)
	}

	if err := validators.IsValidPassword(params.Password); err != nil {
		log.Debug("Invalid password")
		return res, err
	}

	mobile := strings.TrimSpace(params.PhoneNumber)
	if mobile == "" || len(mobile) < 10 {
		log.Debug("Invalid phone number")
		return res, fmt.Errorf("invalid phone number")
	}

	emailInput := strings.ToLower(strings.TrimSpace(refs.StringValue(params.Email)))

	// if email is null set random dummy email for db constraint

	if emailInput != "" && !validators.IsValidEmail(emailInput) {
		log.Debug("Invalid email: ", emailInput)
		return res, fmt.Errorf(`invalid email address`)
	}

	if emailInput == "" {
		emailInput = mobile + "@authorizer.dev"
	}

	log := log.WithFields(log.Fields{
		"email":        emailInput,
		"phone_number": mobile,
	})
	// find user with email
	existingUser, err := db.Provider.GetUserByPhoneNumber(ctx, mobile)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
	}

	if existingUser != nil {
		if existingUser.PhoneNumberVerifiedAt != nil {
			// email is verified
			log.Debug("Phone number is already verified and signed up.")
			return res, fmt.Errorf(`%s has already signed up`, mobile)
		} else if existingUser.ID != "" && existingUser.PhoneNumberVerifiedAt == nil {
			log.Debug("Phone number is already signed up. Verification pending...")
			return res, fmt.Errorf("%s has already signed up. please complete the phone number verification process or reset the password", mobile)
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

	user := models.User{
		Email:       emailInput,
		PhoneNumber: &mobile,
	}

	user.Roles = strings.Join(inputRoles, ",")

	password, _ := crypto.EncryptPassword(params.Password)
	user.Password = &password

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

	disablePhoneVerification, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
	if disablePhoneVerification {
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}

	user.SignupMethods = constants.AuthRecipeMethodMobileBasicAuth
	user, err = db.Provider.AddUser(ctx, user)

	if err != nil {
		log.Debug("Failed to add user: ", err)
		return res, err
	}

	if !disablePhoneVerification {
		duration, _ := time.ParseDuration("10m")
		smsCode := utils.GenerateOTP()

		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(smsCode)

		// TODO: For those who enabled the webhook to call their sms vendor separately - sending the otp to their api
		if err != nil {
			log.Debug("error while upserting user: ", err.Error())
			return nil, err
		}

		go func() {
			db.Provider.UpsertOTP(ctx, &models.OTP{
				PhoneNumber: mobile,
				Otp:         smsCode,
				ExpiresAt:   time.Now().Add(duration).Unix(),
			})
			smsproviders.SendSMS(mobile, smsBody.String())
		}()
	}

	roles := strings.Split(user.Roles, ",")
	userToReturn := user.AsAPIUser()

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

	authToken, err := token.CreateAuthToken(gc, user, roles, scope, constants.AuthRecipeMethodMobileBasicAuth, nonce, code)
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

	sessionKey := constants.AuthRecipeMethodMobileBasicAuth + ":" + user.ID
	cookie.SetSession(gc, authToken.FingerPrintHash)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		utils.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		db.Provider.AddSession(ctx, models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	return res, nil
}
