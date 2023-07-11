package provider_template

import (
	"context"

	"github.com/authorizerdev/authorizer/server/db/models"
)

// UpsertSMSRequest adds/updates SMS verification request
func (p *provider) UpsertSMSRequest(ctx context.Context, smsRequest *models.SMSVerificationRequest) (*models.SMSVerificationRequest, error) {
	return nil, nil
}

// GetCodeByPhone to get code for a given phone number
func (p *provider) GetCodeByPhone(ctx context.Context, phoneNumber string) (*models.SMSVerificationRequest, error) {
	return nil, nil
}

// DeleteSMSRequest to delete SMS verification request
func (p *provider) DeleteSMSRequest(ctx context.Context, smsRequest *models.SMSVerificationRequest) error {
	return nil
}
