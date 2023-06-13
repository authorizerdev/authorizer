package dynamodb

import (
	"context"

	"github.com/authorizerdev/authorizer/server/db/models"

)

// SMS verification Request
func (p *provider) UpsertSMSRequest(ctx context.Context, sms_code *models.SMSVerificationRequest) (*models.SMSVerificationRequest, error) {
	return sms_code, nil
}

func (p *provider) GetCodeByPhone(ctx context.Context, phoneNumber string) (*models.SMSVerificationRequest, error) {
	var sms_verification_request models.SMSVerificationRequest

	return &sms_verification_request, nil
}

func(p *provider) DeleteSMSRequest(ctx context.Context, smsRequest *models.SMSVerificationRequest) error {
	return nil
}
