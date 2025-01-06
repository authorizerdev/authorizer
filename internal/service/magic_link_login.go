package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// MagicLinkLogin is the method to login a user using magic link.
// Permissions: none
func (s *service) MagicLinkLogin(ctx context.Context, params *model.MagicLinkLoginInput) (*model.Response, error) {
	log := s.Log.With().Str("func", "MagicLinkLogin").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	isMagicLinkLoginDisabled := s.Config.DisableMagicLinkLogin
	if isMagicLinkLoginDisabled {
		log.Debug().Msg("Magic link login is disabled")
		return nil, fmt.Errorf(`magic link login is disabled for this instance`)
	}

	params.Email = strings.ToLower(params.Email)
	log = log.With().Str("email", params.Email).Logger()

	if !validators.IsValidEmail(params.Email) {
		log.Debug().Msg("Invalid email address")
		return nil, fmt.Errorf(`invalid email address`)
	}

	inputRoles := []string{}
	user := &schemas.User{
		Email: refs.NewStringRef(params.Email),
	}

	// find user with email
	existingUser, err := s.StorageProvider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		isSignupDisabled := s.Config.DisableSignup
		if isSignupDisabled {
			log.Debug().Msg("Signup is disabled")
			return nil, fmt.Errorf(`signup is disabled for this instance`)
		}

		user.SignupMethods = constants.AuthRecipeMethodMagicLinkLogin
		// define roles for new user
		if len(params.Roles) > 0 {
			// check if roles exists
			roles := strings.Split(s.Config.Roles, ",")
			if !validators.IsValidRoles(params.Roles, roles) {
				log.Debug().Msg("Invalid roles")
				return nil, fmt.Errorf(`invalid roles`)
			} else {
				inputRoles = params.Roles
			}
		} else {
			inputRoles = strings.Split(s.Config.DefaultRoles, ",")
		}

		user.Roles = strings.Join(inputRoles, ",")
		user, _ = s.StorageProvider.AddUser(ctx, user)
		go s.EventsProvider.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodMagicLinkLogin, user)
	} else {
		user = existingUser
		// There multiple scenarios with roles here in magic link login
		// 1. user has access to protected roles + roles and trying to login
		// 2. user has not signed up for one of the available role but trying to signup.
		// 		Need to modify roles in this case

		if user.RevokedTimestamp != nil {
			log.Debug().Msg("User access has been revoked")
			return nil, fmt.Errorf(`user access has been revoked`)
		}

		// find the unassigned roles
		if len(params.Roles) <= 0 {
			inputRolesString := s.DefaultRoles
			inputRoles = strings.Split(inputRolesString, ",")
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
			protectedRolesString := s.Config.ProtectedRoles
			protectedRoles := strings.Split(protectedRolesString, ",")
			for _, ur := range unasignedRoles {
				if utils.StringSliceContains(protectedRoles, ur) {
					hasProtectedRole = true
				}
			}

			if hasProtectedRole {
				log.Debug().Msg("Protected roles cannot be assigned")
				return nil, fmt.Errorf(`invalid roles`)
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
		user, err = s.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Msg("Failed to update user")
			return nil, fmt.Errorf(`failed to update user`)
		}
	}

	hostname := parsers.GetHost(gc)
	isEmailVerificationDisabled := s.Config.DisableEmailVerification
	if !isEmailVerificationDisabled {
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug().Msg("Failed to generate nonce")
			return nil, err
		}
		redirectURLParams := "&roles=" + strings.Join(inputRoles, ",")
		if params.State != nil {
			redirectURLParams = redirectURLParams + "&state=" + refs.StringValue(params.State)
		}
		if len(params.Scope) > 0 {
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
		// params.Email, verificationType, hostname, nonceHash, redirectURL
		verificationToken, err := s.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			User:        user,
			HostName:    hostname,
			Nonce:       nonceHash,
			LoginMethod: constants.AuthRecipeMethodMagicLinkLogin,
		}, redirectURL, verificationType)
		if err != nil {
			log.Debug().Msg("Failed to create verification token")
			return nil, err
		}
		_, err = s.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
			Token:       verificationToken,
			Identifier:  verificationType,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       params.Email,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		})
		if err != nil {
			log.Debug().Msg("Failed to add verification request")
			return nil, err
		}

		// exec it as go routine so that we can reduce the api latency
		go s.EmailProvider.SendEmail([]string{params.Email}, constants.VerificationTypeMagicLinkLogin, map[string]interface{}{
			"user":             user.ToMap(),
			"organization":     utils.GetOrganization(s.Config),
			"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
		})
	}

	return &model.Response{
		Message: `Magic Link has been sent to your email. Please check your inbox!`,
	}, nil
}
