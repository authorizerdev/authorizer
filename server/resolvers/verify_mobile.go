package resolvers

import (
	"fmt"
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/db"
	log "github.com/sirupsen/logrus"
)

func VerifyMobileResolver(ctx context.Context, params model.VerifyMobileRequest) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	_, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	smsVerificationRequest, err := db.Provider.GetCodeByPhone(ctx, params.PhoneNumber)
	if err != nil {
		log.Debug("Failed to get sms request by phone: ", err)
		return res, err
	}

	if smsVerificationRequest.Code != params.Code {
		log.Debug("Failed to verify request: bad credentials")
		return res, fmt.Errorf(`bad credentials`)
	}

	expiresIn := smsVerificationRequest.CodeExpiresAt - time.Now().Unix()
	if expiresIn < 0 {
		log.Debug("Failed to verify sms request: Timeout")
		return res, fmt.Errorf("time expired")
	}

	res = &model.AuthResponse{
		Message: "successful",
	}

	user, err := db.Provider.GetUserByPhoneNumber(ctx, params.PhoneNumber)
	if user.PhoneNumberVerifiedAt == nil {
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}

	_, err = db.Provider.UpdateUser(ctx, *user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}

	err = db.Provider.DeleteSMSRequest(ctx, smsVerificationRequest)
	if err != nil {
		log.Debug("Failed to delete sms request: ", err.Error())
	}

	return res, err
}
