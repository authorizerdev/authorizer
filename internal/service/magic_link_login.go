package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// genericMagicLinkMessage is returned on every non-error path (and on the
// account-revoked path) so a caller cannot probe whether an account exists.
const genericMagicLinkMessage = `If an account exists for this email, a magic link has been sent. Please check your inbox. If you don't receive it within a few minutes, double-check the email address for typos.`

// MagicLinkLogin logs a user in via a magic link, creating the user on the fly
// when signup is enabled. Transport-agnostic port of
// graphqlProvider.MagicLinkLogin.
//
// Permissions: none.
func (p *provider) MagicLinkLogin(ctx context.Context, meta RequestMetadata, params *model.MagicLinkLoginRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "MagicLinkLogin").Logger()

	isMagicLinkLoginEnabled := p.Config.EnableMagicLinkLogin
	if !isMagicLinkLoginEnabled {
		log.Debug().Msg("Magic link login is disabled")
		return nil, nil, FailedPrecondition(`magic link login is disabled for this instance`)
	}

	params.Email = strings.ToLower(params.Email)
	log = log.With().Str("email", params.Email).Logger()

	if !validators.IsValidEmail(params.Email) {
		log.Debug().Msg("Invalid email address")
		return nil, nil, InvalidArgument(`invalid email address`)
	}

	inputRoles := []string{}
	user := &schemas.User{
		Email: refs.NewStringRef(params.Email),
	}

	// find user with email
	existingUser, err := p.StorageProvider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		isSignupEnabled := p.Config.EnableSignup
		if !isSignupEnabled {
			log.Debug().Msg("Signup is disabled")
			return nil, nil, FailedPrecondition(`signup is disabled for this instance`)
		}

		user.SignupMethods = constants.AuthRecipeMethodMagicLinkLogin
		// define roles for new user
		if len(params.Roles) > 0 {
			// check if roles exists
			roles := p.Config.Roles
			if !validators.IsValidRoles(params.Roles, roles) {
				log.Debug().Msg("Invalid roles")
				return nil, nil, InvalidArgument(`invalid roles`)
			} else {
				inputRoles = params.Roles
			}
		} else {
			inputRoles = p.Config.DefaultRoles
		}

		user.Roles = strings.Join(inputRoles, ",")
		user, _ = p.StorageProvider.AddUser(ctx, user)
		asyncutil.Go(p.Log, func() {
			ctx := context.WithoutCancel(ctx)
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserCreatedWebhookEvent, constants.AuthRecipeMethodMagicLinkLogin, user)
		})
	} else {
		user = existingUser
		// There multiple scenarios with roles here in magic link login
		// 1. user has access to protected roles + roles and trying to login
		// 2. user has not signed up for one of the available role but trying to signup.
		// 		Need to modify roles in this case

		if user.RevokedTimestamp != nil {
			// Do not reveal that the account exists but is revoked. Return the
			// same generic "magic link sent" response a successful path would
			// return; the real reason is recorded at debug level.
			log.Debug().Str("reason", "account_revoked").Msg("magic link silently dropped")
			return &model.Response{
				Message: genericMagicLinkMessage,
			}, nil, nil
		}

		// find the unassigned roles
		if len(params.Roles) <= 0 {
			inputRolesString := p.Config.DefaultRoles
			inputRoles = inputRolesString
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
			protectedRoles := p.Config.ProtectedRoles
			for _, ur := range unasignedRoles {
				if utils.StringSliceContains(protectedRoles, ur) {
					hasProtectedRole = true
				}
			}

			if hasProtectedRole {
				log.Debug().Msg("Protected roles cannot be assigned")
				return nil, nil, InvalidArgument(`invalid roles`)
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
		user, err = p.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Msg("Failed to update user")
			return nil, nil, errors.New("failed to update user")
		}
	}

	hostname := meta.HostURL
	isEmailVerificationEnabled := p.Config.EnableEmailVerification
	if isEmailVerificationEnabled {
		// insert verification request
		_, nonceHash, err := utils.GenerateNonce()
		if err != nil {
			log.Debug().Msg("Failed to generate nonce")
			return nil, nil, err
		}
		redirectURLParams := "&roles=" + strings.Join(inputRoles, ",")
		if params.State != nil {
			redirectURLParams = redirectURLParams + "&state=" + refs.StringValue(params.State)
		}
		if len(params.Scope) > 0 {
			redirectURLParams = redirectURLParams + "&scope=" + strings.Join(params.Scope, " ")
		}
		redirectURL := hostname + "/app"
		if params.RedirectURI != nil {
			redirectURL = *params.RedirectURI
			if !validators.IsValidRedirectURI(redirectURL, p.Config.AllowedOrigins, hostname) {
				log.Debug().Msg("Invalid redirect URI")
				return nil, nil, InvalidArgument("invalid redirect URI")
			}
		}

		if strings.Contains(redirectURL, "?") {
			redirectURL = redirectURL + "&" + redirectURLParams
		} else {
			redirectURL = redirectURL + "?" + strings.TrimPrefix(redirectURLParams, "&")
		}

		verificationType := constants.VerificationTypeMagicLinkLogin
		// params.Email, verificationType, hostname, nonceHash, redirectURL
		verificationToken, err := p.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			User:        user,
			HostName:    hostname,
			Nonce:       nonceHash,
			LoginMethod: constants.AuthRecipeMethodMagicLinkLogin,
		}, redirectURL, verificationType)
		if err != nil {
			log.Debug().Msg("Failed to create verification token")
			return nil, nil, err
		}
		_, err = p.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
			Token:       verificationToken,
			Identifier:  verificationType,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       params.Email,
			Nonce:       nonceHash,
			RedirectURI: redirectURL,
		})
		if err != nil {
			log.Debug().Msg("Failed to add verification request")
			return nil, nil, err
		}

		// exec it as go routine so that we can reduce the api latency
		asyncutil.Go(p.Log, func() {
			_ = p.EmailProvider.SendEmail([]string{params.Email}, constants.VerificationTypeMagicLinkLogin, map[string]any{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(p.Config),
				"verification_url": utils.GetEmailVerificationURL(verificationToken, hostname, redirectURL),
			})
		})
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditMagicLinkRequestedEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   params.Email,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: genericMagicLinkMessage,
	}, nil, nil
}
