package dynamodb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/dynamo"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// AddVerification to save verification request in database
func (p *provider) AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error) {
	collection := p.db.Table(schemas.Collections.VerificationRequest)
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
func (p *provider) GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error) {
	collection := p.db.Table(schemas.Collections.VerificationRequest)
	var verificationRequest *schemas.VerificationRequest
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
func (p *provider) GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error) {
	var verificationRequest *schemas.VerificationRequest
	collection := p.db.Table(schemas.Collections.VerificationRequest)
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
	var verificationRequest *schemas.VerificationRequest
	var lastEval dynamo.PagingKey
	var iter dynamo.PagingIter
	var iteration int64 = 0
	collection := p.db.Table(schemas.Collections.VerificationRequest)
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
func (p *provider) DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error {
	collection := p.db.Table(schemas.Collections.VerificationRequest)
	if verificationRequest != nil {
		err := collection.Delete("id", verificationRequest.ID).RunWithContext(ctx)

		if err != nil {
			return err
		}
	}
	return nil
}
