package dynamodb

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/google/uuid"
	"github.com/guregu/dynamo"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *models.VerificationRequest) (*models.VerificationRequest, error) {
	collection := p.db.Table(models.Collections.VerificationRequest)
	if verificationRequest.ID == "" {
		verificationRequest.ID = uuid.New().String()
		verificationRequest.CreatedAt = time.Now().Unix()
		verificationRequest.UpdatedAt = time.Now().Unix()
		err := collection.Put(verificationRequest).RunWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return verificationRequest, nil
}

// GetVerificationRequestByToken to get verification request from database using token
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*models.VerificationRequest, error) {
	collection := p.db.Table(models.Collections.VerificationRequest)
	var verificationRequest *models.VerificationRequest
	iter := collection.Scan().Filter("'token' = ?", token).Iter()
	for iter.NextWithContext(ctx, &verificationRequest) {
		return verificationRequest, nil
	}
	err := iter.Err()
	if err != nil {
		return nil, err
	}
	return verificationRequest, nil
}

// GetVerificationRequestByEmail to get verification request by email from database
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*models.VerificationRequest, error) {
	var verificationRequest *models.VerificationRequest
	collection := p.db.Table(models.Collections.VerificationRequest)
	iter := collection.Scan().Filter("'email' = ?", email).Filter("'identifier' = ?", identifier).Iter()
	for iter.NextWithContext(ctx, &verificationRequest) {
		return verificationRequest, nil
	}
	err := iter.Err()
	if err != nil {
		return nil, err
	}
	return verificationRequest, nil
}

// ListVerificationRequests to get list of verification requests from database
func (p *provider) ListVerificationRequests(ctx context.Context, pagination *model.Pagination) (*model.VerificationRequests, error) {
	verificationRequests := []*model.VerificationRequest{}
	var verificationRequest *models.VerificationRequest
	var lastEval dynamo.PagingKey
	var iter dynamo.PagingIter
	var iteration int64 = 0
	collection := p.db.Table(models.Collections.VerificationRequest)
	paginationClone := pagination
	scanner := collection.Scan()
	count, err := scanner.Count()
	if err != nil {
		return nil, err
	}
	for (paginationClone.Offset + paginationClone.Limit) > iteration {
		iter = scanner.StartFrom(lastEval).Limit(paginationClone.Limit).Iter()
		for iter.NextWithContext(ctx, &verificationRequest) {
			if paginationClone.Offset == iteration {
				verificationRequests = append(verificationRequests, verificationRequest.AsAPIVerificationRequest())
			}
		}
		err = iter.Err()
		if err != nil {
			return nil, err
		}
		lastEval = iter.LastEvaluatedKey()
		iteration += paginationClone.Limit
	}
	paginationClone.Total = count
	return &model.VerificationRequests{
		VerificationRequests: verificationRequests,
		Pagination:           paginationClone,
	}, nil
}

// DeleteVerificationRequest to delete verification request from database
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *models.VerificationRequest) error {
	collection := p.db.Table(models.Collections.VerificationRequest)
	if verificationRequest != nil {
		err := collection.Delete("id", verificationRequest.ID).RunWithContext(ctx)

		if err != nil {
			return err
		}
	}
	return nil
}
