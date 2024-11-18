package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/data_store/db"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// MagicLinkLoginResolver is a resolver for magic link login mutation
func MagicLinkLoginResolver(ctx context.Context, params model.MagicLinkLoginInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	isMagicLinkLoginDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMagicLinkLogin)
	if err != nil {
		log.Debug("Error getting magic link login disabled: ", err)
		isMagicLinkLoginDisabled = true
	}

	if isMagicLinkLoginDisabled {
		log.Debug("Magic link login is disabled.")
		return res, fmt.Errorf(`magic link login is disabled for this instance`)
	}

	params.Email = strings.ToLower(params.Email)

	if !validators.IsValidEmail(params.Email) {
		log.Debug("Invalid email")
		return res, fmt.Errorf(`invalid email address`)
	}

	log := log.WithFields(log.Fields{
		"email": params.Email,
	})

	inputRoles := []string{}

	user := &models.User{
		Email: refs.NewStringRef(params.Email),
	}

	// find user with email
	existingUser, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		isSignupDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableSignUp)
		if err != nil {
			log.Debug("Error getting signup disabled: ", err)
		}
		if isSignupDisabled {
			log.Debug("Signup is disabled.")
			return res, fmt.Errorf(`signup is disabled for this instance`)
		}

		user.SignupMethods = constants.AuthRecipeMethodMagicLinkLogin
		// define roles for new user
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
				return res, fmt.Errorf(`invalid roles`)
			} else {
				inputRoles = strings.Split(inputRolesString, ",")
			}
		}

		user.Roles = strings.Join(inputRoles, ",")
		user, _ = db.Provider.AddUser(ctx, user)
		go utils.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodMagicLinkLogin, user)
	} else {
		user = existingUser
		// There multiple scenarios with roles here in magic link login
		// 1. user has access to protected roles + roles and trying to login
		// 2. user has not signed up for one of the available role but trying to signup.
		// 		Need to modify roles in this case

		if user.RevokedTimestamp != nil {
			log.Debug("User access is revoked at: ", user.RevokedTimestamp)
			return res, fmt.Errorf(`user access has been revoked`)
		}

		// find the unassigned roles
		if len(params.Roles) <= 0 {
			inputRolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
			if err != nil {
				log.Debug("Error getting default roles: ", err)
				return res, fmt.Errorf(`invalid default roles`)
			} else {
				inputRoles = strings.Split(inputRolesString, ",")
			}
		}
		existingRoles := strings.Split(existingUser.Roles, ",")
		unasignedRoles := []string{}
		for _, ir := range inputRoles {
			if !utils.StringSliceContains(existingRoles, ir) {
				unasignedRoles = append(unasignedRoles, ir)
			}
		}

		if len(unasignedRoles) > 0 {
			// check if it contains protected unassigned role
			hasProtectedRole := false
			protectedRolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyProtectedRoles)
			protectedRoles := []string{}
			if err != nil {
				log.Debug("Error getting protected roles: ", err)
				return res, err
			} else {
				protectedRoles = strings.Split(protectedRolesString, ",")
			}
			for _, ur := range unasignedRoles {
				if utils.StringSliceContains(protectedRoles, ur) {
					hasProtectedRole = true
				}
			}

			if hasProtectedRole {
				log.Debug("User is not assigned one of the protected roles", unasignedRoles)
				return res, fmt.Errorf(`invalid roles`)
			} else {
				user.Roles = existingUser.Roles + "," + strings.Join(unasignedRoles, ",")
			}
		} else {
			user.Roles = existingUser.Roles
		}

		signupMethod := existingUser.SignupMethods
		if !strings.Contains(signupMethod, constants.AuthRecipeMethodMagicLinkLogin) {
			signupMethod = signupMethod + "," + constants.AuthRecipeMethodMagicLinkLogin
		}

		user.SignupMethods = signupMethod
		user, _ = db.Provider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug("Failed to update user: ", err)
		}
	}

	hostname := parsers.GetHost(gc)
	isEmailVerificationDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification)
	if err != nil {
		log.Debug("Error getting email verification disabled: ", err)
		isEmailVerificationDisabled = true
	}
	if !isEmailVerificationDisabled {
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug("Failed to generate nonce: ", err)
			return res, err
		}
		redirectURLParams := "&roles=" + strings.Join(inputRoles, ",")
		if params.State != nil {
			redirectURLParams = redirectURLParams + "&state=" + refs.StringValue(params.State)
		}
		if params.Scope != nil && len(params.Scope) > 0 {
			redirectURLParams = redirectURLParams + "&scope=" + strings.Join(params.Scope, " ")
		}
		redirectURL := parsers.GetAppURL(gc)
		if params.RedirectURI != nil {
			redirectURL = *params.RedirectURI
		}

		if strings.Contains(redirectURL, "?") {
			redirectURL = redirectURL + "&" + redirectURLParams
		} else {
			redirectURL = redirectURL + "?" + strings.TrimPrefix(redirectURLParams, "&")
		}

		verificationType := constants.VerificationTypeMagicLinkLogin
		verificationToken, err := token.CreateVerificationToken(params.Email, verificationType, hostname, nonceHash, redirectURL)
		if err != nil {
			log.Debug("Failed to create verification token: ", err)
		}
		_, err = db.Provider.AddVerificationRequest(ctx, &models.VerificationRequest{
			Token:       verificationToken,
			Identifier:  verificationType,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       params.Email,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		})
		if err != nil {
			log.Debug("Failed to add verification request in db: ", err)
			return res, err
		}

		// exec it as go routine so that we can reduce the api latency
		go email.SendEmail([]string{params.Email}, constants.VerificationTypeMagicLinkLogin, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(),
			"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
		})
	}

	res = &model.Response{
		Message: `Magic Link has been sent to your email. Please check your inbox!`,
	}

	return res, nil
}
