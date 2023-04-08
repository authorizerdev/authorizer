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
	"github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
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

	if isBasicAuthDisabled {
		log.Debug("Basic authentication is disabled")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	if params.ConfirmPassword != params.Password {
		log.Debug("Passwords do not match")
		return res, fmt.Errorf(`password and confirm password does not match`)
	}

	if err := validators.IsValidPassword(params.Password); err != nil {
		log.Debug("Invalid password")
		return res, err
	}

	params.Email = strings.ToLower(params.Email)

	if !validators.IsValidEmail(params.Email) {
		log.Debug("Invalid email: ", params.Email)
		return res, fmt.Errorf(`invalid email address`)
	}

	log := log.WithFields(log.Fields{
		"email": params.Email,
	})
	// find user with email
	existingUser, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
	}

	if existingUser.EmailVerifiedAt != nil {
		// email is verified
		log.Debug("Email is already verified and signed up.")
		return res, fmt.Errorf(`%s has already signed up`, params.Email)
	} else if existingUser.ID != "" && existingUser.EmailVerifiedAt == nil {
		log.Debug("Email is already signed up. Verification pending...")
		return res, fmt.Errorf("%s has already signed up. please complete the email verification process or reset the password", params.Email)
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
		Email: params.Email,
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

	if params.PhoneNumber != nil {
		user.PhoneNumber = params.PhoneNumber
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

	user.SignupMethods = constants.AuthRecipeMethodBasicAuth
	isEmailVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification)
	if err != nil {
		log.Debug("Error getting email verification disabled: ", err)
		isEmailVerificationDisabled = true
	}
	if isEmailVerificationDisabled {
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}
	user, err = db.Provider.AddUser(ctx, user)
	if err != nil {
		log.Debug("Failed to add user: ", err)
		return res, err
	}
	roles := strings.Split(user.Roles, ",")
	userToReturn := user.AsAPIUser()

	hostname := parsers.GetHost(gc)
	if !isEmailVerificationDisabled {
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
		verificationToken, err := token.CreateVerificationToken(params.Email, verificationType, hostname, nonceHash, redirectURL)
		if err != nil {
			log.Debug("Failed to create verification token: ", err)
			return res, err
		}
		_, err = db.Provider.AddVerificationRequest(ctx, models.VerificationRequest{
			Token:       verificationToken,
			Identifier:  verificationType,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       params.Email,
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
			email.SendEmail([]string{params.Email}, constants.VerificationTypeBasicAuthSignup, map[string]interface{}{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(),
				"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname),
			})
			utils.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()

		res = &model.AuthResponse{
			Message: `Verification email has been sent. Please check your inbox`,
			User:    userToReturn,
		}
	} else {
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
			utils.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
			db.Provider.AddSession(ctx, models.Session{
				UserID:    user.ID,
				UserAgent: utils.GetUserAgent(gc.Request),
				IP:        utils.GetIP(gc.Request),
			})
		}()
	}

	return res, nil
}
