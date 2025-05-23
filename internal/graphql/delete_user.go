package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// DeleteUser is the method to delete a user.
// Permissions: authorizer:admin
func (g *graphqlProvider) DeleteUser(ctx context.Context, params *model.DeleteUserRequest) (*model.Response, error) {
	log := g.Log.With().Str("func", "DeleteUser").Logger()

	var res *model.Response
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return res, err
	}

	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	log = log.With().Str("email", params.Email).Logger()
	user, err := g.StorageProvider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by email")
		return res, err
	}

	err = g.StorageProvider.DeleteUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete user")
		return res, err
	}

	res = &model.Response{
		Message: `user deleted successfully`,
	}

	go func() {
		// delete otp for given email
		otp, err := g.StorageProvider.GetOTPByEmail(ctx, refs.StringValue(user.Email))
		if err != nil {
			log.Debug().Err(err).Msg("No OTP found for email")
			// continue
		} else {
			err := g.StorageProvider.DeleteOTP(ctx, otp)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to delete otp for given email")
				// continue
			}
		}

		// delete otp for given phone number
		otp, err = g.StorageProvider.GetOTPByPhoneNumber(ctx, refs.StringValue(user.PhoneNumber))
		if err != nil {
			log.Debug().Err(err).Msg("No OTP found for phone number")
			// continue
		} else {
			err := g.StorageProvider.DeleteOTP(ctx, otp)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to delete otp for given phone number")
				// continue
			}
		}

		// delete verification requests for given email
		for _, vt := range constants.VerificationTypes {
			verificationRequest, err := g.StorageProvider.GetVerificationRequestByEmail(ctx, refs.StringValue(user.Email), vt)
			if err != nil {
				log.Debug().Err(err).Msg("No verification request found for email")
				// continue
			} else {
				err := g.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest)
				if err != nil {
					log.Debug().Err(err).Msg("Failed to delete verification request for given email")
					// continue
				}
			}
		}

		g.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
		g.EventsProvider.RegisterEvent(ctx, constants.UserDeletedWebhookEvent, "", user)
	}()

	return res, nil
}
