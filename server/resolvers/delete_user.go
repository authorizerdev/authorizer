package resolvers

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// DeleteUserResolver is a resolver for delete user mutation
func DeleteUserResolver(ctx context.Context, params model.DeleteUserInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return res, fmt.Errorf("unauthorized")
	}

	log := log.WithFields(log.Fields{
		"email": params.Email,
	})

	user, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("Failed to get user from DB: ", err)
		return res, err
	}

	err = db.Provider.DeleteUser(ctx, user)
	if err != nil {
		log.Debug("Failed to delete user: ", err)
		return res, err
	}

	res = &model.Response{
		Message: `user deleted successfully`,
	}

	go func() {
		// delete otp for given email
		otp, err := db.Provider.GetOTPByEmail(ctx, user.Email)
		if err != nil {
			log.Debugf("Failed to get otp for given email (%s): %v", user.Email, err)
			// continue
		} else {
			err := db.Provider.DeleteOTP(ctx, otp)
			if err != nil {
				log.Debugf("Failed to delete otp for given email (%s): %v", user.Email, err)
				// continue
			}
		}

		// delete verification requests for given email
		for _, vt := range constants.VerificationTypes {
			verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, user.Email, vt)
			if err != nil {
				log.Debug("Failed to get verification request for email: %s, verification_request_type: %s. %v", user.Email, vt, err)
				// continue
			} else {
				err := db.Provider.DeleteVerificationRequest(ctx, verificationRequest)
				if err != nil {
					log.Debug("Failed to DeleteVerificationRequest for email: %s, verification_request_type: %s. %v", user.Email, vt, err)
					// continue
				}
			}
		}

		memorystore.Provider.DeleteAllUserSessions(user.ID)
		utils.RegisterEvent(ctx, constants.UserDeletedWebhookEvent, "", user)
	}()

	return res, nil
}
