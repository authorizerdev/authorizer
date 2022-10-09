package couchbase

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest models.VerificationRequest) (models.VerificationRequest, error) {
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
	}

	verificationRequest.CreatedAt = time.Now().Unix()
	verificationRequest.UpdatedAt = time.Now().Unix()

	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (models.VerificationRequest, error) {
	var verificationRequest models.VerificationRequest

	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (models.VerificationRequest, error) {
	var verificationRequest models.VerificationRequest

	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination model.Pagination) (*model.VerificationRequests, error) {
	return nil, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest models.VerificationRequest) error {
	return nil
}
